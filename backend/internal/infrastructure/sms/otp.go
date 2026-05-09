package sms

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Rate limiting constants
const (
	// Maximum OTP requests per phone number per time window
	MaxOTPRequests = 3
	// Time window for rate limiting (5 minutes)
	RateLimitWindow = 5 * time.Minute
	// Minimum time between OTP requests (1 minute)
	MinRequestInterval = 1 * time.Minute
)

type OTPStore struct {
	redisClient *redis.Client
	apiKey      string
	lineNumber  string
	httpClient  *http.Client
}

// SMS.ir API request/response structures
type smsirSendRequest struct {
	LineNumber  string   `json:"lineNumber"`
	MessageText string   `json:"messageText"`
	Mobiles     []string `json:"mobiles"`
}

type smsirResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// RateLimitError represents a rate limit violation
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// NewOTPStore creates a new OTP store with Redis backend
func NewOTPStore(redisAddr, apiKey, lineNumber string) *OTPStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password by default
		DB:       0,  // default DB
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Printf("[WARNING] Redis connection failed: %v. OTP will not work!\n", err)
	}

	return &OTPStore{
		redisClient: rdb,
		apiKey:      apiKey,
		lineNumber:  lineNumber,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// GenerateOTP generates a new OTP code and stores it in Redis with 2-minute TTL
func (s *OTPStore) GenerateOTP(phone string) (string, error) {
	ctx := context.Background()

	// Check rate limits
	if err := s.checkRateLimit(ctx, phone); err != nil {
		return "", err
	}

	code, err := generateNumericOTP(6)
	if err != nil {
		return "", err
	}

	// Store in Redis with 2-minute expiration
	key := fmt.Sprintf("otp:%s", phone)
	err = s.redisClient.Set(ctx, key, code, 2*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store OTP in Redis: %w", err)
	}

	// Send SMS
	if err := s.sendSMS(phone, code); err != nil {
		// If SMS fails, delete the OTP from Redis
		s.redisClient.Del(ctx, key)
		return "", fmt.Errorf("failed to send SMS: %w", err)
	}

	// Increment rate limit counter
	s.incrementRateLimit(ctx, phone)

	return code, nil
}

// checkRateLimit checks if the phone number has exceeded rate limits
func (s *OTPStore) checkRateLimit(ctx context.Context, phone string) error {
	// Check 1: Minimum interval between requests (1 minute)
	lastRequestKey := fmt.Sprintf("otp:last:%s", phone)
	lastRequestTime, err := s.redisClient.Get(ctx, lastRequestKey).Result()
	if err == nil {
		// Last request time exists
		lastTime, _ := strconv.ParseInt(lastRequestTime, 10, 64)
		timeSinceLastRequest := time.Since(time.Unix(lastTime, 0))
		if timeSinceLastRequest < MinRequestInterval {
			retryAfter := MinRequestInterval - timeSinceLastRequest
			return &RateLimitError{
				RetryAfter: retryAfter,
				Message:    fmt.Sprintf("Please wait %d seconds before requesting another OTP", int(retryAfter.Seconds())),
			}
		}
	}

	// Check 2: Maximum requests per time window (3 requests per 5 minutes)
	rateLimitKey := fmt.Sprintf("otp:rate:%s", phone)
	count, err := s.redisClient.Get(ctx, rateLimitKey).Int()
	if err != nil && err != redis.Nil {
		// Redis error, log but don't block
		fmt.Printf("[WARNING] Rate limit check failed: %v\n", err)
		return nil
	}

	if count >= MaxOTPRequests {
		ttl, _ := s.redisClient.TTL(ctx, rateLimitKey).Result()
		return &RateLimitError{
			RetryAfter: ttl,
			Message:    fmt.Sprintf("Too many OTP requests. Please try again in %d minutes", int(ttl.Minutes())+1),
		}
	}

	return nil
}

// incrementRateLimit increments the rate limit counters
func (s *OTPStore) incrementRateLimit(ctx context.Context, phone string) {
	// Update last request time
	lastRequestKey := fmt.Sprintf("otp:last:%s", phone)
	s.redisClient.Set(ctx, lastRequestKey, time.Now().Unix(), RateLimitWindow)

	// Increment request counter
	rateLimitKey := fmt.Sprintf("otp:rate:%s", phone)
	pipe := s.redisClient.Pipeline()
	pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, RateLimitWindow)
	pipe.Exec(ctx)
}

// sendSMS sends OTP via SMS.ir API or prints to console in mock mode
func (s *OTPStore) sendSMS(phone, code string) error {
	// Mock mode: if no API key, just log
	if s.apiKey == "" {
		fmt.Printf("[MOCK SMS] Sending OTP %s to %s\n", code, phone)
		return nil
	}

	// Normalize phone number for SMS.ir
	// SMS.ir expects: 09xxxxxxxxx (without +98 prefix)
	normalizedPhone := normalizePhoneForSMSIR(phone)

	message := fmt.Sprintf("کد تایید شما: %s\nاکادمی نوشیروانی", code)

	reqBody := smsirSendRequest{
		LineNumber:  s.lineNumber,
		MessageText: message,
		Mobiles:     []string{normalizedPhone},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.sms.ir/v1/send/bulk", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS.ir API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var smsResp smsirResponse
	if err := json.Unmarshal(body, &smsResp); err != nil {
		return err
	}
	println(smsResp)
	if smsResp.Status != 1 {
		return fmt.Errorf("SMS.ir returned error: %s", smsResp.Message)
	}

	return nil
}

// VerifyOTP checks if the provided code matches the stored OTP in Redis
func (s *OTPStore) VerifyOTP(phone, code string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", phone)

	// Get OTP from Redis
	storedCode, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// OTP not found or expired
		return false
	} else if err != nil {
		fmt.Printf("[ERROR] Redis error during OTP verification: %v\n", err)
		return false
	}

	// Check if code matches
	if storedCode == code {
		// Delete OTP after successful verification (one-time use)
		s.redisClient.Del(ctx, key)
		return true
	}

	return false
}

// normalizePhoneForSMSIR converts phone number to SMS.ir format
// Accepts: +989123456789, 989123456789, 09123456789
// Returns: 09123456789
func normalizePhoneForSMSIR(phone string) string {
	// Remove + prefix
	phone = strings.TrimPrefix(phone, "+")

	// Remove 98 country code if present
	if strings.HasPrefix(phone, "98") {
		phone = "0" + phone[2:]
	}

	// Ensure it starts with 0
	if !strings.HasPrefix(phone, "0") {
		phone = "0" + phone
	}

	return phone
}

func generateNumericOTP(length int) (string, error) {
	const nums = "0123456789"
	result := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, result); err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		result[i] = nums[int(result[i])%len(nums)]
	}
	return string(result), nil
}
