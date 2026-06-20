package tests

import (
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
)

// GET /health
func TestHealth(t *testing.T) {
	resp := do(t, http.MethodGet, "/health", "", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
	}
	var body map[string]string
	resp.JSON(t, &body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body)
	}
}

// GET /subjects
func TestGetSubjectsByMajor(t *testing.T) {
	t.Run("valid major", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/subjects?major=%D8%B1%DB%8C%D8%A7%D8%B6%DB%8C", "", nil) // ریاضی
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var cfg struct {
			Major    string   `json:"major"`
			Subjects []string `json:"subjects"`
		}
		resp.JSON(t, &cfg)
		if len(cfg.Subjects) == 0 {
			t.Fatalf("expected subjects for valid major, got none")
		}
	})

	t.Run("missing major param", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/subjects", "", nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("invalid major", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/subjects?major=nonsense", "", nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// GET /majors
func TestGetAllMajors(t *testing.T) {
	resp := do(t, http.MethodGet, "/majors", "", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
	}
	var majors []struct {
		Major    string   `json:"major"`
		Subjects []string `json:"subjects"`
	}
	resp.JSON(t, &majors)
	if len(majors) != 4 {
		t.Fatalf("expected 4 majors, got %d", len(majors))
	}
}

// GET /blog (public list) and GET /blog/:slug (public get)
func TestPublicBlog(t *testing.T) {
	resetDB(t)

	// Seed one published and one unpublished post.
	published := domain.BlogPost{Title: "Hello World", Slug: "hello-world", Content: "body", Published: true}
	draft := domain.BlogPost{Title: "Draft", Slug: "draft", Content: "body", Published: false}
	if err := testDB.Create(&published).Error; err != nil {
		t.Fatalf("seed published: %v", err)
	}
	if err := testDB.Create(&draft).Error; err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	t.Run("public list returns only published", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/blog", "", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var posts []domain.BlogPost
		resp.JSON(t, &posts)
		if len(posts) != 1 || posts[0].Slug != "hello-world" {
			t.Fatalf("expected only published post, got %+v", posts)
		}
	})

	t.Run("public get by slug", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/blog/hello-world", "", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("public get unpublished -> 404", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/blog/draft", "", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for unpublished, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("public get missing slug -> 404", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/blog/does-not-exist", "", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body)
		}
	})
}
