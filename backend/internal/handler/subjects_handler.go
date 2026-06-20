package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SubjectsHandler struct{}

// SubjectConfig پیکربندی دروس یک رشته
type SubjectConfig struct {
	Major    string   `json:"major" description:"نام رشته"`
	Subjects []string `json:"subjects" description:"لیست دروس"`
}

func NewSubjectsHandler() *SubjectsHandler {
	return &SubjectsHandler{}
}

// GetSubjectsByMajor godoc
// @Summary دریافت دروس بر اساس رشته
// @Description لیست دروس مرتبط با یک رشته تحصیلی را دریافت می‌کند
// @Tags دروس
// @Produce json
// @Param major query string true "نام رشته (ریاضی، تجربی، انسانی)"
// @Success 200 {object} SubjectConfig "لیست دروس"
// @Failure 400 {object} ErrorResponse "رشته نامعتبر"
// @Router /subjects [get]
func (h *SubjectsHandler) GetSubjectsByMajor(c *gin.Context) {
	major := c.Query("major")
	if major == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "major parameter is required"})
		return
	}

	subjects := getSubjectsForMajor(major)
	if subjects == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid major"})
		return
	}

	c.JSON(http.StatusOK, SubjectConfig{
		Major:    major,
		Subjects: subjects,
	})
}

// GetAllMajors godoc
// @Summary دریافت تمام رشته‌ها
// @Description لیست تمام رشته‌های موجود را دریافت می‌کند
// @Tags دروس
// @Produce json
// @Success 200 {array} SubjectConfig "لیست رشته‌ها و دروس"
// @Router /majors [get]
func (h *SubjectsHandler) GetAllMajors(c *gin.Context) {
	majors := []SubjectConfig{
		{
			Major:    "ریاضی",
			Subjects: getSubjectsForMajor("ریاضی"),
		},
		{
			Major:    "تجربی",
			Subjects: getSubjectsForMajor("تجربی"),
		},
		{
			Major:    "انسانی",
			Subjects: getSubjectsForMajor("انسانی"),
		},
		{
			Major:    "هنر",
			Subjects: getSubjectsForMajor("هنر"),
		},
	}

	c.JSON(http.StatusOK, majors)
}

// getSubjectsForMajor returns the list of subjects for a given major
func getSubjectsForMajor(major string) []string {
	subjectsMap := map[string][]string{
		"ریاضی": {
			"ریاضی",
			"فیزیک",
			"شیمی",
			"زبان انگلیسی",
			"ادبیات فارسی",
			"عربی",
			"دین و زندگی",
			"زمین‌شناسی",
		},
		"تجربی": {
			"ریاضی",
			"فیزیک",
			"شیمی",
			"زیست‌شناسی",
			"زبان انگلیسی",
			"ادبیات فارسی",
			"عربی",
			"دین و زندگی",
		},
		"انسانی": {
			"ادبیات فارسی",
			"عربی",
			"زبان انگلیسی",
			"دین و زندگی",
			"تاریخ",
			"جغرافیا",
			"فلسفه و منطق",
			"روانشناسی و علوم تربیتی",
			"اقتصاد",
		},
		"هنر": {
			"ادبیات فارسی",
			"زبان انگلیسی",
			"دین و زندگی",
			"هنر",
			"ریاضی",
			"تاریخ هنر",
			"طراحی",
		},
	}

	return subjectsMap[major]
}
