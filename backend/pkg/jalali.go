package pkg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yaa110/go-persian-calendar"
)

func GregorianToJalaliString(t time.Time) string {
	p := ptime.New(t)
	return p.Format("yyyy/MM/dd")
}

func JalaliToGregorian(jalaliDate string) (time.Time, error) {
	parts := strings.Split(jalaliDate, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format, expected yyyy/MM/dd")
	}

	year, err1 := strconv.Atoi(parts[0])
	month, err2 := strconv.Atoi(parts[1])
	day, err3 := strconv.Atoi(parts[2])

	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, fmt.Errorf("invalid date numbers")
	}

	p := ptime.Date(year, ptime.Month(month), day, 0, 0, 0, 0, ptime.Iran())
	return p.Time(), nil
}
