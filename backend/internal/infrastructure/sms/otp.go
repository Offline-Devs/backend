package sms

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type OTPStore struct {
	redisClient *redis.Client
	apiKey     string
	lineNumber string
	httpClient *http.Client
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
		apiKey:     apiKey,
		lineNumber: lineNumber,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GenerateOTP generates a new OTP code and stores it in Redis with 2-minute TTL
func (s *OTPStore) GenerateOTP(phone string) (string, error) {
	code, err := generateNumericOTP(6)
	if err != nil {
		return "", err
	}

	// Store in Redis with 2-minute expiration
	ctx := context.Background()
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

	return code, nil
}

// sendSMS sends OTP via SMS.ir API or prints to console in mock mode
func (s *OTPStore) sendSMS(phone, code string) error {
	// Mock mode: if no API key, just log
	if s.apiKey == "" {
		fmt.Printf("[MOCK SMS] Sending OTP %s to %s\n", code, phone)
		return nil
	}

	message := fmt.Sprintf("کد تایید شما: %s\nاکادمی نوشیروانی", code)

	reqBody := smsirSendRequest{
		LineNumber:  s.lineNumber,
		MessageText: message,
		Mobiles:     []string{phone},
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
