package tests

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// doUpload performs a multipart/form-data POST with the given files.
// files maps the form field name -> (filename -> content).
func doUpload(t *testing.T, path, token string, fieldName string, files map[string][]byte) apiResponse {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for name, content := range files {
		fw, err := w.CreateFormFile(fieldName, name)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.RemoteAddr = fmt.Sprintf("11.%d.%d.%d:12345",
		atomic.AddUint32(&ipCounter, 1)%256, ipCounter%256, ipCounter%251+1)

	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	return apiResponse{Code: rec.Code, Body: rec.Body.Bytes()}
}

// POST /upload
func TestUploadFile(t *testing.T) {
	resetDB(t)
	_, _, token := createPendingStudent(t)

	t.Run("pending student blocked", func(t *testing.T) {
		resp := doUpload(t, "/upload?type=profile", token, "file", map[string][]byte{
			"avatar.png": []byte("\x89PNG\r\n\x1a\nfake-image-bytes"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("disallowed extension -> 400", func(t *testing.T) {
		resp := doUpload(t, "/upload", token, "file", map[string][]byte{
			"malware.exe": []byte("MZ..."),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("no file -> 400", func(t *testing.T) {
		resp := doUpload(t, "/upload", token, "wrongfield", map[string][]byte{
			"x.png": []byte("data"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("no auth -> 401", func(t *testing.T) {
		resp := doUpload(t, "/upload", "", "file", map[string][]byte{"x.png": []byte("d")})
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("path traversal upload type -> 400", func(t *testing.T) {
		resp := doUpload(t, "/upload?type=../../tmp/pwn", token, "file", map[string][]byte{
			"avatar.png": []byte("png"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("approved student can upload", func(t *testing.T) {
		_, _, approvedToken := createStudent(t)
		resp := doUpload(t, "/upload?type=profile", approvedToken, "file", map[string][]byte{
			"avatar.png": []byte("\x89PNG\r\n\x1a\nfake-image-bytes"),
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("admin can upload any document extension", func(t *testing.T) {
		_, adminToken := createAdmin(t)
		resp := doUpload(t, "/upload?type=document", adminToken, "file", map[string][]byte{
			"archive.zip": []byte("zip-content"),
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// POST /upload/multiple
func TestUploadMultiple(t *testing.T) {
	resetDB(t)
	_, _, token := createPendingStudent(t)

	t.Run("pending student blocked", func(t *testing.T) {
		resp := doUpload(t, "/upload/multiple", token, "files", map[string][]byte{
			"a.png": []byte("img-a"),
			"b.pdf": []byte("pdf-b"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("all invalid -> 400", func(t *testing.T) {
		resp := doUpload(t, "/upload/multiple", token, "files", map[string][]byte{
			"a.exe": []byte("x"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("no files -> 400", func(t *testing.T) {
		resp := doUpload(t, "/upload/multiple", token, "wrongfield", map[string][]byte{
			"a.png": []byte("x"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("invalid upload type -> 400", func(t *testing.T) {
		resp := doUpload(t, "/upload/multiple?type=../../tmp/pwn", token, "files", map[string][]byte{
			"a.png": []byte("x"),
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("approved student multiple upload works", func(t *testing.T) {
		_, _, approvedToken := createStudent(t)
		resp := doUpload(t, "/upload/multiple", approvedToken, "files", map[string][]byte{
			"a.png": []byte("img-a"),
			"b.pdf": []byte("pdf-b"),
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
	})
}
