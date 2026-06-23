package tests

import (
	"errors"
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

// createExam is a helper that POSTs an exam and returns its id.
func createExam(t *testing.T, token, title string) string {
	t.Helper()
	resp := do(t, http.MethodPost, "/exams", token, map[string]interface{}{
		"title":         title,
		"jalali_date":   "1403/03/03",
		"major":         "ریاضی",
		"negative_mark": 0.25,
		"subjects": []map[string]interface{}{
			{"subject_name": "ریاضی", "total_questions": 10, "correct": 7, "wrong": 3},
		},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create exam expected 201, got %d: %s", resp.Code, resp.Body)
	}
	var e domain.Exam
	resp.JSON(t, &e)
	if e.ID == "" {
		t.Fatalf("expected exam id, got %s", resp.Body)
	}
	return e.ID
}

// POST/GET/PUT/DELETE /exams
func TestExamCRUD(t *testing.T) {
	resetDB(t)
	_, _, token := createStudent(t)

	var examID string

	t.Run("create", func(t *testing.T) {
		examID = createExam(t, token, "Midterm")
	})

	t.Run("create persists subjects", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/exams/"+examID, token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var e domain.Exam
		resp.JSON(t, &e)
		if len(e.Subjects) != 1 || e.Subjects[0].Correct != 7 {
			t.Fatalf("subjects not persisted: %+v", e.Subjects)
		}
		if e.NegativeMark != 0.25 {
			t.Fatalf("negative mark not persisted: %v", e.NegativeMark)
		}
	})

	t.Run("list", func(t *testing.T) {
		createExam(t, token, "Quiz")
		resp := do(t, http.MethodGet, "/exams", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var exams []domain.Exam
		resp.JSON(t, &exams)
		if len(exams) != 2 {
			t.Fatalf("expected 2 exams, got %d", len(exams))
		}
	})

	t.Run("get one", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/exams/"+examID, token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("get missing -> 404", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/exams/00000000-0000-0000-0000-000000000000", token, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("update title and subjects", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/exams/"+examID, token, map[string]interface{}{
			"title":         "Updated",
			"negative_mark": 0.33,
			"subjects": []map[string]interface{}{
				{"subject_name": "فیزیک", "total_questions": 20, "correct": 15, "wrong": 5},
			},
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var e domain.Exam
		resp.JSON(t, &e)
		if e.Title != "Updated" {
			t.Fatalf("title not updated: %+v", e)
		}
		if e.NegativeMark != 0.33 {
			t.Fatalf("negative mark not updated: %v", e.NegativeMark)
		}
		if len(e.Subjects) != 1 || e.Subjects[0].SubjectName != "فیزیک" {
			t.Fatalf("subjects not replaced: %+v", e.Subjects)
		}
	})

	t.Run("update invalid jalali date -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/exams/"+examID, token, map[string]interface{}{
			"jalali_date": "garbage",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("create rejects mixed gregorian and jalali input", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/exams", token, map[string]interface{}{
			"title":       "Ambiguous",
			"exam_date":   "2026-01-01T00:00:00Z",
			"jalali_date": "1404/10/11",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("create canonicalizes jalali date", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/exams", token, map[string]interface{}{
			"title":       "Canonical",
			"jalali_date": "1403/9/5",
		})
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body)
		}
		var e domain.Exam
		resp.JSON(t, &e)
		if e.JalaliDate != "1403/09/05" {
			t.Fatalf("expected canonical jalali date, got %q", e.JalaliDate)
		}
	})

	t.Run("update invalid negative mark -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/exams/"+examID, token, map[string]interface{}{
			"negative_mark": 1.1,
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("update missing exam -> 404", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/exams/00000000-0000-0000-0000-000000000000", token, map[string]interface{}{
			"title": "x",
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("delete", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/exams/"+examID, token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		// Confirm it is gone.
		get := do(t, http.MethodGet, "/exams/"+examID, token, nil)
		if get.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", get.Code)
		}
	})

	t.Run("delete missing exam -> 404", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/exams/00000000-0000-0000-0000-000000000000", token, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// CreateExam without a student profile must 404.
func TestExamWithoutProfile(t *testing.T) {
	resetDB(t)
	_, token := createUser(t, "student")
	resp := do(t, http.MethodPost, "/exams", token, map[string]interface{}{"title": "x"})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (no profile), got %d: %s", resp.Code, resp.Body)
	}
}

// A student must not be able to read another student's exam.
func TestExamCrossStudentIsolation(t *testing.T) {
	resetDB(t)
	_, _, tokenA := createStudent(t)
	_, _, tokenB := createStudent(t)

	examID := createExam(t, tokenA, "A's exam")

	t.Run("other student gets 404 on GET", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/exams/"+examID, tokenB, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("other student gets 404 on UPDATE", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/exams/"+examID, tokenB, map[string]interface{}{"title": "hijack"})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})
}

func TestExamUpdateRollbackOnSubjectFailure(t *testing.T) {
	resetDB(t)
	_, _, token := createStudent(t)
	examID := createExam(t, token, "Rollback")

	callbackName := "tests:fail_subject_exam_create"
	err := testDB.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Schema != nil && tx.Statement.Schema.Table == "subject_exams" {
			tx.AddError(errors.New("forced subject create failure"))
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}
	defer func() {
		_ = testDB.Callback().Create().Remove(callbackName)
	}()

	resp := do(t, http.MethodPut, "/exams/"+examID, token, map[string]interface{}{
		"title": "Should Roll Back",
		"subjects": []map[string]interface{}{
			{"subject_name": "فیزیک", "total_questions": 20, "correct": 15, "wrong": 5},
		},
	})
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", resp.Code, resp.Body)
	}

	get := do(t, http.MethodGet, "/exams/"+examID, token, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("expected 200 on reload, got %d: %s", get.Code, get.Body)
	}
	var exam domain.Exam
	get.JSON(t, &exam)
	if exam.Title != "Rollback" {
		t.Fatalf("expected title rollback, got %q", exam.Title)
	}
	if len(exam.Subjects) != 1 || exam.Subjects[0].SubjectName != "ریاضی" {
		t.Fatalf("expected original subjects preserved, got %+v", exam.Subjects)
	}
}
