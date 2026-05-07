package sms

import (
    "crypto/rand"
    "io"
    "sync"
    "time"
)

type OTPStore struct {
    mu    sync.Mutex
    store map[string]otpEntry
}

type otpEntry struct {
    Code      string
    ExpiresAt time.Time
}

func NewOTPStore() *OTPStore {
    return &OTPStore{store: make(map[string]otpEntry)}
}

func (s *OTPStore) GenerateOTP(phone string) (string, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    otp := generateNumericOTP(6)
    s.store[phone] = otpEntry{
        Code:      otp,
        ExpiresAt: time.Now().Add(2 * time.Minute),
    }
    return otp, nil
}

func (s *OTPStore) VerifyOTP(phone, code string) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    entry, ok := s.store[phone]
    if !ok {
        return false
    }
    if time.Now().After(entry.ExpiresAt) {
        delete(s.store, phone)
        return false
    }
    if entry.Code == code {
        delete(s.store, phone)
        return true
    }
    return false
}

func generateNumericOTP(length int) string {
    const nums = "0123456789"
    result := make([]byte, length)
    if _, err := io.ReadFull(rand.Reader, result); err != nil {
        panic(err)
    }
    for i := 0; i < length; i++ {
        result[i] = nums[int(result[i])%len(nums)]
    }
    return string(result)
}
