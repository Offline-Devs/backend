package pkg

import (
	"fmt"
	"strings"
	"time"

	ptime "github.com/yaa110/go-persian-calendar"
)

func GregorianToJalaliString(t time.Time) string {
	pt := ptime.New(t)
	return pt.Format("yyyy/MM/dd")
}

func JalaliToGregorian(jalaliDate string) (time.Time, error) {
	jalaliDate = strings.TrimSpace(jalaliDate)
	if jalaliDate == "" {
		return time.Time{}, fmt.Errorf("empty jalali date")
	}

	var year, month, day int
	n, err := fmt.Sscanf(jalaliDate, "%d/%d/%d", &year, &month, &day)
	if err != nil || n != 3 {
		return time.Time{}, fmt.Errorf("invalid jalali format")
	}
	if year < 1200 || year > 1700 {
		return time.Time{}, fmt.Errorf("invalid jalali year")
	}
	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("invalid jalali month")
	}
	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("invalid jalali day")
	}

	pt := ptime.Date(year, ptime.Month(month), day, 0, 0, 0, 0, ptime.Iran())
	if pt.Year() != year || int(pt.Month()) != month || pt.Day() != day {
		return time.Time{}, fmt.Errorf("invalid jalali date")
	}
	return pt.Time(), nil
}
