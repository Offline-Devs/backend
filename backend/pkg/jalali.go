package pkg

import (
	"fmt"
	"time"

	ptime "github.com/yaa110/go-persian-calendar"
)

func GregorianToJalaliString(t time.Time) string {
	pt := ptime.New(t)
	return pt.Format("yyyy/MM/dd")
}

func JalaliToGregorian(jalaliDate string) (time.Time, error) {
	// این پکیج Parse ندارد، باید manual parsing کنید
	// یا از پکیج دیگری استفاده کنید
	var year, month, day int
	_, err := fmt.Sscanf(jalaliDate, "%d/%d/%d", &year, &month, &day)
	if err != nil {
		return time.Time{}, err
	}

	pt := ptime.Date(year, ptime.Month(month), day, 0, 0, 0, 0, ptime.Iran())
	return pt.Time(), nil
}
