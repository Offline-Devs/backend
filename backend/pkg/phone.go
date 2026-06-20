package pkg

import (
	"strings"
)

// NormalizePhone converts different phone formats to a standard format
// Supports:
// - 09123456789 (leading zero)
// - +989123456789 (with country code)
// - 989123456789 (without plus sign)
// Returns format: 989123456789 (without leading 0 or +)
func NormalizePhone(phone string) string {
	if phone == "" {
		return ""
	}

	// Remove whitespace
	phone = strings.TrimSpace(phone)

	// Remove plus sign
	phone = strings.TrimPrefix(phone, "+")

	// Replace leading 0 with 98 (Iran country code)
	if strings.HasPrefix(phone, "0") {
		phone = "98" + phone[1:]
	}

	return phone
}
