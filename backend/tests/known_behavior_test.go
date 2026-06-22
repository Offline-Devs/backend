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

// ISSUE #1: mutating/deleting a non-existent resource returns 200 success
// instead of 404, because handlers issue UPDATE/DELETE without checking that a
// row actually matched.
func TestBehavior_MissingResourceReturnsSuccess(t *testing.T) {
	resetDB(t)
	_, adminToken := createAdmin(t)
	missing := "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		method, path string
	}{
		{http.MethodPut, "/admin/students/" + missing},
		{http.MethodPut, "/admin/students/" + missing + "/approve"},
		{http.MethodDelete, "/admin/students/" + missing},
		{http.MethodDelete, "/admin/blog/" + missing},
		{http.MethodPut, "/admin/blog/" + missing},
		{http.MethodPut, "/admin/performance/" + missing},
		{http.MethodDelete, "/admin/performance/" + missing},
		{http.MethodDelete, "/admin/dynamic-fields/" + missing},
	}
	for _, tc := range cases {
		resp := do(t, tc.method, tc.path, adminToken, map[string]interface{}{"city": "x", "notes": "x", "title": "x"})
		if resp.Code != http.StatusOK {
			t.Errorf("%s %s: expected current behaviour 200, got %d (issue may be fixed)", tc.method, tc.path, resp.Code)
		}
	}

	// DELETE /exams/{missing} also returns 200 — but only once the caller has a
	// student profile (the handler checks the profile first, then deletes 0 rows).
	_, _, studentToken := createStudent(t)
	resp := do(t, http.MethodDelete, "/exams/"+missing, studentToken, nil)
	if resp.Code != http.StatusOK {
		t.Errorf("DELETE /exams/missing with profile: expected current behaviour 200, got %d", resp.Code)
	}
}

// ISSUE #2: a deactivated user (is_active=false) can still mint fresh access
// tokens via /auth/refresh — RefreshToken never checks IsActive, while
// VerifyOTP does.
func TestBehavior_InactiveUserCanRefresh(t *testing.T) {
	resetDB(t)
	u := domain.User{Phone: uniquePhone(), Role: "student", IsActive: true}
	if err := testDB.Create(&u).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	testDB.Exec("UPDATE users SET is_active=false WHERE id=?", u.ID)

	refresh, _ := jwtSvc.GenerateRefreshToken(u.ID)
	resp := do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": refresh})
	if resp.Code != http.StatusOK {
		t.Errorf("expected current behaviour 200 (inactive user refresh allowed), got %d", resp.Code)
	}
}

// ISSUE #3: a student can attach another student's exam_id to their own
// mistake. There is no ownership check on the referenced exam_id.
func TestBehavior_MistakeAcceptsForeignExamID(t *testing.T) {
	resetDB(t)
	_, _, tokenA := createStudent(t)
	_, studentB, _ := createStudent(t)
	examB := domain.Exam{StudentID: studentB, Title: "B"}
	if err := testDB.Create(&examB).Error; err != nil {
		t.Fatalf("seed exam: %v", err)
	}

	resp := do(t, http.MethodPost, "/mistakes", tokenA, map[string]interface{}{
		"question_number": 1, "exam_id": examB.ID,
	})
	if resp.Code != http.StatusCreated {
		t.Errorf("expected current behaviour 201 (foreign exam_id accepted), got %d", resp.Code)
	}
}

// ISSUE #4: client-supplied Jalali dates are now canonicalized to zero-padded
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
