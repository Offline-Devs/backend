package phone

import "strings"

// Normalize converts supported Iranian mobile formats into 09xxxxxxxxx.
func Normalize(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "+")

	if strings.HasPrefix(value, "98") {
		value = "0" + value[2:]
	}

	if len(value) == 10 && strings.HasPrefix(value, "9") {
		value = "0" + value
	}

	return value
}
