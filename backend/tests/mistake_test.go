package tests

import (
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
)

// POST/GET/PUT/DELETE /mistakes
func TestMistakeCRUD(t *testing.T) {
	resetDB(t)
	_, _, token := createStudent(t)

	var mistakeID string

	t.Run("create", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/mistakes", token, map[string]interface{}{
			"question_number": 5,
			"category":        "carelessness",
			"notes":           "forgot the formula",
		})
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body)
		}
		var m domain.Mistake
		resp.JSON(t, &m)
		if m.ID == "" || m.QuestionNumber != 5 {
			t.Fatalf("unexpected mistake: %+v", m)
		}
		mistakeID = m.ID
	})

	t.Run("create with non-positive question_number -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/mistakes", token, map[string]interface{}{
			"question_number": 0,
			"category":        "x",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("list", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/mistakes", token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var mistakes []domain.Mistake
		resp.JSON(t, &mistakes)
		if len(mistakes) != 1 {
			t.Fatalf("expected 1 mistake, got %d", len(mistakes))
		}
	})

	t.Run("update", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/mistakes/"+mistakeID, token, map[string]interface{}{
			"category": "conceptual",
			"notes":    "did not understand",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var m domain.Mistake
		resp.JSON(t, &m)
		if m.Category != "conceptual" {
			t.Fatalf("category not updated: %+v", m)
		}
	})

	t.Run("update with non-positive question_number -> 400", func(t *testing.T) {
		bad := -1
		resp := do(t, http.MethodPut, "/mistakes/"+mistakeID, token, map[string]interface{}{
			"question_number": bad,
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("update missing mistake -> 404", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/mistakes/00000000-0000-0000-0000-000000000000", token, map[string]interface{}{
			"category": "x",
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("delete", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/mistakes/"+mistakeID, token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		list := do(t, http.MethodGet, "/mistakes", token, nil)
		var mistakes []domain.Mistake
		list.JSON(t, &mistakes)
		if len(mistakes) != 0 {
			t.Fatalf("expected 0 mistakes after delete, got %d", len(mistakes))
		}
	})
}

// Create without a profile must 404.
func TestMistakeWithoutProfile(t *testing.T) {
	resetDB(t)
	_, token := createUser(t, "student")
	resp := do(t, http.MethodPost, "/mistakes", token, map[string]interface{}{"question_number": 1})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (no profile), got %d: %s", resp.Code, resp.Body)
	}
}

func TestMistakeRejectsForeignReferences(t *testing.T) {
	resetDB(t)
	_, studentA, tokenA := createStudent(t)
	_, studentB, _ := createStudent(t)
	examB := domain.Exam{StudentID: studentB, Title: "B"}
	if err := testDB.Create(&examB).Error; err != nil {
		t.Fatalf("seed exam: %v", err)
	}
	subjectB := domain.SubjectExam{ExamID: examB.ID, SubjectName: "Math"}
	if err := testDB.Create(&subjectB).Error; err != nil {
		t.Fatalf("seed subject: %v", err)
	}

	t.Run("create rejects foreign exam_id", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/mistakes", tokenA, map[string]interface{}{
			"question_number": 1,
			"exam_id":         examB.ID,
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("create rejects foreign subject_exam_id", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/mistakes", tokenA, map[string]interface{}{
			"question_number": 1,
			"subject_exam_id": subjectB.ID,
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	ownedExam := domain.Exam{StudentID: studentA, Title: "A"}
	if err := testDB.Create(&ownedExam).Error; err != nil {
		t.Fatalf("seed owned exam: %v", err)
	}
	ownedMistake := domain.Mistake{StudentID: studentA, QuestionNumber: 2}
	if err := testDB.Create(&ownedMistake).Error; err != nil {
		t.Fatalf("seed mistake: %v", err)
	}

	t.Run("update rejects foreign exam_id", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/mistakes/"+ownedMistake.ID, tokenA, map[string]interface{}{
			"exam_id": examB.ID,
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("update rejects mismatched subject for owned exam", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/mistakes/"+ownedMistake.ID, tokenA, map[string]interface{}{
			"exam_id":         ownedExam.ID,
			"subject_exam_id": subjectB.ID,
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})
}
