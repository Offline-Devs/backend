package pkg

import (
    "time"

    calendar "github.com/yaa110/go-persian-calendar"
)

func GregorianToJalaliString(t time.Time) string {
    p := calendar.NewPersian(t)
    return p.String()
}

func JalaliToGregorian(jalaliDate string) (time.Time, error) {
    p, err := calendar.Parse(jalaliDate)
    if err != nil {
        return time.Time{}, err
    }
    return p.Time(), nil
}
