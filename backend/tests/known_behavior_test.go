package tests

// These tests document CURRENTLY OBSERVED behaviour that is arguably wrong or
// surprising. They are written to PASS against today's code so they double as
// regression markers: if someone fixes the underlying issue, the corresponding
// test will fail and should be updated. Each issue is described in
// ENDPOINT_TEST_REPORT.md.

import (
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
)

// ISSUE #1: client-supplied Jalali dates are now canonicalized to zero-padded
// form so storage and range filtering stay consistent.
func TestBehavior_RawJalaliDateStored(t *testing.T) {
	resetDB(t)
	_, sid, token := createStudent(t)

	resp := do(t, http.MethodPost, "/exams", token, map[string]interface{}{
		"title": "x", "jalali_date": "1403/9/5",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create exam: %d %s", resp.Code, resp.Body)
	}

	var e domain.Exam
	if err := testDB.Where("student_id = ?", sid).First(&e).Error; err != nil {
		t.Fatalf("reload exam: %v", err)
	}
	if e.JalaliDate != "1403/09/05" {
		t.Errorf("expected canonical padded date stored, got %q", e.JalaliDate)
	}
}
