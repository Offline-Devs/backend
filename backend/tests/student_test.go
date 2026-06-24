package tests

import (
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
)

// POST /students/profile  &  GET /students/profile
func TestStudentProfile(t *testing.T) {
	resetDB(t)
	// A user without a student profile yet.
	_, token := createUser(t, "student")

	t.Run("get profile before creation -> 404", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/profile", token, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("create profile", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/students/profile", token, map[string]interface{}{
			"first_name":        "Ali",
			"last_name":         "Rezaei",
			"city":              "Tehran",
			"major":             "ریاضی",
			"jalali_birth_date": "1380/05/15",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var s domain.Student
		resp.JSON(t, &s)
		if s.FirstName != "Ali" || s.ID == "" {
			t.Fatalf("unexpected student: %+v", s)
		}
		if s.BirthDate.IsZero() {
			t.Fatalf("expected jalali birth date to be converted to a gregorian date")
		}
	})

	t.Run("update existing profile", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/students/profile", token, map[string]interface{}{
			"first_name": "Ali",
			"last_name":  "Mohammadi",
			"city":       "Shiraz",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var s domain.Student
		resp.JSON(t, &s)
		if s.LastName != "Mohammadi" || s.City != "Shiraz" {
			t.Fatalf("update not applied: %+v", s)
		}
	})

	t.Run("get profile after creation", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/profile", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("missing names -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/students/profile", token, map[string]interface{}{
			"first_name": "   ",
			"last_name":  "",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("invalid jalali date -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/students/profile", token, map[string]interface{}{
			"first_name":        "A",
			"last_name":         "B",
			"jalali_birth_date": "not-a-date",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("rejects mixed gregorian and jalali birth dates", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/students/profile", token, map[string]interface{}{
			"first_name":        "Ali",
			"last_name":         "Rezaei",
			"birth_date":        "2001-08-06T00:00:00Z",
			"jalali_birth_date": "1380/05/15",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("canonicalizes jalali birth date", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/students/profile", token, map[string]interface{}{
			"first_name":        "Ali",
			"last_name":         "Rezaei",
			"jalali_birth_date": "1380/5/15",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var s domain.Student
		resp.JSON(t, &s)
		if s.JalaliBirthDate != "1380/05/15" {
			t.Fatalf("expected canonical jalali date, got %q", s.JalaliBirthDate)
		}
	})

	t.Run("no auth -> 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/profile", "", nil)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// GET /students/performance
func TestStudentPerformance(t *testing.T) {
	resetDB(t)
	_, studentID, token := createStudent(t)

	// Seed a performance record (normally created by an admin).
	if err := testDB.Create(&domain.PerformanceHistory{
		StudentID: studentID, JalaliDate: "1403/01/01", Notes: "n", StudyPlan: "plan",
	}).Error; err != nil {
		t.Fatalf("seed performance: %v", err)
	}

	t.Run("returns records", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/performance", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var recs []domain.PerformanceHistory
		resp.JSON(t, &recs)
		if len(recs) != 1 {
			t.Fatalf("expected 1 record, got %d", len(recs))
		}
	})

	t.Run("no profile -> 404", func(t *testing.T) {
		_, noProfileToken := createUser(t, "student")
		resp := do(t, http.MethodGet, "/students/performance", noProfileToken, nil)
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("pending student -> 403", func(t *testing.T) {
		_, _, pendingToken := createPendingStudent(t)
		resp := do(t, http.MethodGet, "/students/performance", pendingToken, nil)
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// GET /students/statistics  &  GET /students/dashboard
func TestStudentStatisticsAndDashboard(t *testing.T) {
	resetDB(t)
	_, studentID, token := createStudent(t)

	// One exam with two subjects: 8/10 + 6/10 correct -> 70% average.
	exam := domain.Exam{
		StudentID: studentID, Title: "Exam 1", JalaliDate: "1403/02/02",
		Subjects: []domain.SubjectExam{
			{SubjectName: "ریاضی", TotalQuestions: 10, Correct: 8, Wrong: 2},
			{SubjectName: "فیزیک", TotalQuestions: 10, Correct: 6, Wrong: 4},
		},
	}
	if err := testDB.Create(&exam).Error; err != nil {
		t.Fatalf("seed exam: %v", err)
	}
	if err := testDB.Create(&domain.Mistake{StudentID: studentID, QuestionNumber: 1, Category: "carelessness"}).Error; err != nil {
		t.Fatalf("seed mistake: %v", err)
	}

	t.Run("statistics", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/statistics", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var stats struct {
			TotalExams       int64          `json:"total_exams"`
			AverageScore     float64        `json:"average_score"`
			MistakesByReason map[string]int `json:"mistakes_by_reason"`
		}
		resp.JSON(t, &stats)
		if stats.TotalExams != 1 {
			t.Fatalf("expected 1 exam, got %d", stats.TotalExams)
		}
		if stats.AverageScore < 69.9 || stats.AverageScore > 70.1 {
			t.Fatalf("expected ~70%% average, got %v", stats.AverageScore)
		}
		if stats.MistakesByReason["carelessness"] != 1 {
			t.Fatalf("expected mistake category counted, got %+v", stats.MistakesByReason)
		}
	})

	t.Run("statistics with date filter excluding exam", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/statistics?from=1404/01/01", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var stats struct {
			TotalExams int64 `json:"total_exams"`
		}
		resp.JSON(t, &stats)
		if stats.TotalExams != 0 {
			t.Fatalf("expected exam filtered out, got %d", stats.TotalExams)
		}
	})

	t.Run("statistics rejects invalid jalali filter", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/statistics?from=invalid", token, nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("dashboard", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/dashboard", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var summary map[string]interface{}
		resp.JSON(t, &summary)
		if summary["total_exams"].(float64) != 1 {
			t.Fatalf("expected total_exams=1, got %v", summary["total_exams"])
		}
		if summary["total_mistakes"].(float64) != 1 {
			t.Fatalf("expected total_mistakes=1, got %v", summary["total_mistakes"])
		}
	})

	t.Run("statistics no profile -> 404", func(t *testing.T) {
		_, noProfileToken := createUser(t, "student")
		resp := do(t, http.MethodGet, "/students/statistics", noProfileToken, nil)
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("pending student blocked from dashboard", func(t *testing.T) {
		_, _, pendingToken := createPendingStudent(t)
		resp := do(t, http.MethodGet, "/students/dashboard", pendingToken, nil)
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})
}
