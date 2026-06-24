package sms

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	phoneutil "github.com/yourusername/noshirvani-academy/backend/internal/phone"
)

// Rate limiting constants
const (
	// Maximum OTP requests per phone number per time window
	MaxOTPRequests = 3
	// Time window for rate limiting (5 minutes)
	RateLimitWindow = 5 * time.Minute
	// Minimum time between OTP requests (1 minute)
	MinRequestInterval = 1 * time.Minute
	// Maximum failed OTP verification attempts before temporary lock.
	MaxOTPVerifyAttempts = 5
	otpTTL               = 2 * time.Minute
)

type OTPStore struct {
	redisClient *redis.Client
	provider    string
	apiKey      string
	templateID  string
	httpClient  *http.Client
}

// SMS.ir Template API request structure
type smsirVerifyRequest struct {
	Mobile     string           `json:"mobile"`
	TemplateID int              `json:"templateId"`
	Parameters []smsirParameter `json:"parameters"`
}

type smsirParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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

type VerifyLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *VerifyLimitError) Error() string {
	return e.Message
}

// NewOTPStore creates a new OTP store with Redis backend and SMS.ir template
func NewOTPStore(redisAddr, provider, apiKey, templateID string) *OTPStore {
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
		provider:    provider,
		apiKey:      apiKey,
		templateID:  templateID,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// GenerateOTP generates a new OTP code and stores it in Redis with 2-minute TTL
func (s *OTPStore) GenerateOTP(phone string) (string, error) {
	ctx := context.Background()
	phone = NormalizePhone(phone)

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
	err = s.redisClient.Set(ctx, key, code, otpTTL).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store OTP in Redis: %w", err)
	}
	if err := s.resetVerifyState(ctx, phone); err != nil {
		return "", fmt.Errorf("failed to reset otp verify state: %w", err)
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

// sendSMS sends OTP via SMS.ir Template API or prints to console in mock mode
func (s *OTPStore) sendSMS(phone, code string) error {
	// Mock mode deliberately logs the code instead of contacting the SMS provider.
	if s.provider == "mock" {
		fmt.Printf("[MOCK SMS] Sending OTP %s to %s\n", code, phone)
		return nil
	}

	// Normalize phone number for SMS.ir
	// SMS.ir expects: 09xxxxxxxxx (without +98 prefix)
	normalizedPhone := normalizePhoneForSMSIR(phone)

	// Parse template ID
	templateIDInt, err := strconv.Atoi(s.templateID)
	if err != nil {
		return fmt.Errorf("invalid template ID: %w", err)
	}

	// Build request with template
	reqBody := smsirVerifyRequest{
		Mobile:     normalizedPhone,
		TemplateID: templateIDInt,
		Parameters: []smsirParameter{
			{
				Name:  "CODE",
				Value: code,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// Use verify endpoint for template-based sending
	req, err := http.NewRequest("POST", "https://api.sms.ir/v1/send/verify", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
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

	if smsResp.Status != 1 {
		return fmt.Errorf("SMS.ir returned error: %s", smsResp.Message)
	}

	return nil
}

// VerifyOTP checks if the provided code matches the stored OTP in Redis
func (s *OTPStore) VerifyOTP(phone, code string) (bool, error) {
	ctx := context.Background()
	phone = NormalizePhone(phone)
	if err := s.checkVerifyLimit(ctx, phone); err != nil {
		return false, err
	}
	key := fmt.Sprintf("otp:%s", phone)

	// Get OTP from Redis
	storedCode, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// OTP not found or expired
		return false, nil
	} else if err != nil {
		fmt.Printf("[ERROR] Redis error during OTP verification: %v\n", err)
		return false, nil
	}

	// Check if code matches
	if storedCode == code {
		// Delete OTP after successful verification (one-time use)
		s.redisClient.Del(ctx, key)
		if err := s.resetVerifyState(ctx, phone); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := s.incrementVerifyAttempt(ctx, phone); err != nil {
		return false, err
	}

	return false, nil
}

func NormalizePhone(raw string) string {
	return phoneutil.Normalize(raw)
}

func normalizePhoneForSMSIR(phone string) string {
	return NormalizePhone(phone)
}

func (s *OTPStore) checkVerifyLimit(ctx context.Context, phone string) error {
	lockKey := fmt.Sprintf("otp:lock:%s", phone)
	locked, err := s.redisClient.Exists(ctx, lockKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		fmt.Printf("[WARNING] OTP verify lock check failed: %v\n", err)
		return nil
	}
	if locked == 0 {
		return nil
	}

	ttl, _ := s.redisClient.TTL(ctx, lockKey).Result()
	if ttl <= 0 {
		ttl = otpTTL
	}
	return &VerifyLimitError{
		RetryAfter: ttl,
		Message:    fmt.Sprintf("Too many invalid OTP attempts. Please try again in %d seconds", int(ttl.Seconds())+1),
	}
}

func (s *OTPStore) incrementVerifyAttempt(ctx context.Context, phone string) error {
	attemptKey := fmt.Sprintf("otp:attempt:%s", phone)
	lockKey := fmt.Sprintf("otp:lock:%s", phone)
	otpKey := fmt.Sprintf("otp:%s", phone)

	ttl, err := s.redisClient.TTL(ctx, otpKey).Result()
	if err != nil || ttl <= 0 {
		ttl = otpTTL
	}

	count, err := s.redisClient.Incr(ctx, attemptKey).Result()
	if err != nil {
		return err
	}
	if err := s.redisClient.Expire(ctx, attemptKey, ttl).Err(); err != nil {
		return err
	}
	if count < MaxOTPVerifyAttempts {
		return nil
	}
	return s.redisClient.Set(ctx, lockKey, 1, ttl).Err()
}

func (s *OTPStore) resetVerifyState(ctx context.Context, phone string) error {
	return s.redisClient.Del(
		ctx,
		fmt.Sprintf("otp:attempt:%s", phone),
		fmt.Sprintf("otp:lock:%s", phone),
	).Err()
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
