package tests

import (
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
)

// RBAC: a student token must be rejected by /admin routes.
func TestAdminRBAC(t *testing.T) {
	resetDB(t)
	_, _, studentToken := createStudent(t)

	resp := do(t, http.MethodGet, "/admin/students", studentToken, nil)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for student on admin route, got %d: %s", resp.Code, resp.Body)
	}

	noauth := do(t, http.MethodGet, "/admin/students", "", nil)
	if noauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no token, got %d: %s", noauth.Code, noauth.Body)
	}
}

// GET /admin/students  &  GET /admin/students/with-stats
func TestAdminListStudents(t *testing.T) {
	resetDB(t)
	_, adminToken := createAdmin(t)
	createStudent(t)
	createStudent(t)

	t.Run("list", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var students []domain.Student
		resp.JSON(t, &students)
		if len(students) != 2 {
			t.Fatalf("expected 2 students, got %d", len(students))
		}
	})

	t.Run("with-stats pagination", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/with-stats?page=1&limit=1", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var body struct {
			Data  []map[string]interface{} `json:"data"`
			Total int64                    `json:"total"`
			Page  int                      `json:"page"`
			Limit int                      `json:"limit"`
		}
		resp.JSON(t, &body)
		if body.Total != 2 || len(body.Data) != 1 || body.Limit != 1 {
			t.Fatalf("unexpected pagination: total=%d data=%d limit=%d", body.Total, len(body.Data), body.Limit)
		}
		if _, ok := body.Data[0]["exam_count"]; !ok {
			t.Fatalf("expected exam_count field in stats, got %+v", body.Data[0])
		}
	})

	t.Run("with-stats approved filter", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/with-stats?approved=true", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var body struct {
			Total int64 `json:"total"`
		}
		resp.JSON(t, &body)
		if body.Total != 0 {
			t.Fatalf("expected 0 approved students, got %d", body.Total)
		}
	})
}

// GET/PUT/DELETE /admin/students/:id and related sub-resources.
func TestAdminStudentManagement(t *testing.T) {
	resetDB(t)
	_, adminToken := createAdmin(t)
	_, studentID, studentToken := createStudent(t)

	// Seed an exam and a mistake for the student.
	createExam(t, studentToken, "Seeded exam")
	if err := testDB.Create(&domain.Mistake{StudentID: studentID, QuestionNumber: 3, Category: "x"}).Error; err != nil {
		t.Fatalf("seed mistake: %v", err)
	}

	t.Run("get student", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/"+studentID, adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("get missing student -> 404", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/00000000-0000-0000-0000-000000000000", adminToken, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("get student exams", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/"+studentID+"/exams", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var exams []domain.Exam
		resp.JSON(t, &exams)
		if len(exams) != 1 {
			t.Fatalf("expected 1 exam, got %d", len(exams))
		}
	})

	t.Run("get student mistakes", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/"+studentID+"/mistakes", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var mistakes []domain.Mistake
		resp.JSON(t, &mistakes)
		if len(mistakes) != 1 {
			t.Fatalf("expected 1 mistake, got %d", len(mistakes))
		}
	})

	t.Run("get student statistics", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/"+studentID+"/statistics", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("update student", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/students/"+studentID, adminToken, map[string]interface{}{
			"city": "Isfahan",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var reloaded domain.Student
		if err := testDB.First(&reloaded, "id = ?", studentID).Error; err != nil {
			t.Fatalf("reload student: %v", err)
		}
		if reloaded.City != "Isfahan" {
			t.Fatalf("city not updated, got %q", reloaded.City)
		}
	})

	t.Run("update with no fields -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/students/"+studentID, adminToken, map[string]interface{}{})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("approve student", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/students/"+studentID+"/approve", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var reloaded domain.Student
		testDB.First(&reloaded, "id = ?", studentID)
		if !reloaded.IsApproved {
			t.Fatalf("student not approved")
		}
	})

	t.Run("delete student", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/admin/students/"+studentID, adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// Admin performance endpoints.
func TestAdminPerformance(t *testing.T) {
	resetDB(t)
	_, adminToken := createAdmin(t)
	_, studentID, _ := createStudent(t)

	var perfID string

	t.Run("create performance", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/admin/students/"+studentID+"/performance", adminToken, map[string]interface{}{
			"jalali_date": "1403/04/04",
			"notes":       "good progress",
			"study_plan":  "chapter 5",
		})
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body)
		}
		var p domain.PerformanceHistory
		resp.JSON(t, &p)
		if p.ID == "" {
			t.Fatalf("expected id, got %s", resp.Body)
		}
		if p.JalaliDate != "1403/04/04" {
			t.Fatalf("expected canonical jalali date, got %q", p.JalaliDate)
		}
		perfID = p.ID
	})

	t.Run("create performance for missing student -> 404", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/admin/students/00000000-0000-0000-0000-000000000000/performance", adminToken, map[string]interface{}{
			"notes": "x",
		})
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("list performance", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/students/"+studentID+"/performance", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var recs []domain.PerformanceHistory
		resp.JSON(t, &recs)
		if len(recs) != 1 {
			t.Fatalf("expected 1 record, got %d", len(recs))
		}
	})

	t.Run("update performance", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/performance/"+perfID, adminToken, map[string]interface{}{
			"notes": "updated note",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("delete performance", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/admin/performance/"+perfID, adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// Admin dynamic-field endpoints.
func TestAdminDynamicFields(t *testing.T) {
	resetDB(t)
	_, adminToken := createAdmin(t)

	var fieldID string

	t.Run("create", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/admin/dynamic-fields", adminToken, map[string]interface{}{
			"entity_type": "student",
			"name":        "guardian_phone",
			"label":       "Guardian Phone",
			"field_type":  "text",
		})
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body)
		}
		var f domain.DynamicFieldDefinition
		resp.JSON(t, &f)
		fieldID = f.ID
	})

	t.Run("create missing required fields -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/admin/dynamic-fields", adminToken, map[string]interface{}{
			"label": "no entity/name/type",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("list with entity filter", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/dynamic-fields?entity_type=student", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var fields []domain.DynamicFieldDefinition
		resp.JSON(t, &fields)
		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
	})

	t.Run("update", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/dynamic-fields/"+fieldID, adminToken, map[string]interface{}{
			"entity_type": "student",
			"name":        "guardian_phone",
			"field_type":  "number",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("delete", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/admin/dynamic-fields/"+fieldID, adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// Admin blog endpoints.
func TestAdminBlog(t *testing.T) {
	resetDB(t)
	_, adminToken := createAdmin(t)

	var postID string

	t.Run("create", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/admin/blog", adminToken, map[string]interface{}{
			"title":   "My First Post",
			"content": "hello",
		})
		// NOTE: handler returns 200 (not 201) on create.
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var p domain.BlogPost
		resp.JSON(t, &p)
		if p.Slug != "my-first-post" {
			t.Fatalf("expected auto slug, got %q", p.Slug)
		}
		postID = p.ID
	})

	t.Run("create missing title -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/admin/blog", adminToken, map[string]interface{}{
			"content": "no title",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("admin list shows unpublished", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/admin/blog", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var posts []domain.BlogPost
		resp.JSON(t, &posts)
		if len(posts) != 1 {
			t.Fatalf("expected 1 post, got %d", len(posts))
		}
	})

	t.Run("update", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/blog/"+postID, adminToken, map[string]interface{}{
			"title":   "Renamed Post",
			"content": "updated",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("publish then visible publicly", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/admin/blog/"+postID+"/publish", adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		pub := do(t, http.MethodGet, "/blog", "", nil)
		var posts []domain.BlogPost
		pub.JSON(t, &posts)
		if len(posts) != 1 {
			t.Fatalf("expected published post to appear publicly, got %d", len(posts))
		}
	})

	t.Run("delete", func(t *testing.T) {
		resp := do(t, http.MethodDelete, "/admin/blog/"+postID, adminToken, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})
}
