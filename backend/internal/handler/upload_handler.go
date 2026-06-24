package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	uploadPath string
}

var allowedUploadTypes = map[string]string{
	"document": "document",
	"profile":  "profile",
}

// UploadResponse پاسخ آپلود فایل
type UploadResponse struct {
	URL      string `json:"url" description:"آدرس دسترسی به فایل"`
	Filename string `json:"filename" description:"نام فایل"`
	Size     int64  `json:"size" description:"حجم فایل (بایت)"`
}

func NewUploadHandler(uploadPath string) *UploadHandler {
	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		fmt.Printf("[WARNING] Failed to create upload directory: %v\n", err)
	}
	return &UploadHandler{uploadPath: uploadPath}
}

func normalizeUploadType(fileType string) (string, bool) {
	normalized, ok := allowedUploadTypes[strings.TrimSpace(strings.ToLower(fileType))]
	return normalized, ok
}

// generateUniqueFilename generates a unique filename using timestamp and random string
func generateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	// Use cryptographically secure random bytes to prevent predictability
	randBytes := make([]byte, 12)
	if _, err := rand.Read(randBytes); err != nil {
		// fallback to timestamp-based name if crypto fails (very unlikely)
		return fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), time.Now().Unix(), ext)
	}
	return fmt.Sprintf("%s%s", hex.EncodeToString(randBytes), ext)
}

// UploadFile godoc
// @Summary آپلود فایل
// @Description آپلود تصویر پروفایل یا فایل‌های مستندات
// @Tags آپلود
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "فایل برای آپلود"
// @Param type query string false "نوع فایل (profile, document)" default(document)
// @Success 200 {object} UploadResponse "فایل با موفقیت آپلود شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر یا فایل خیلی بزرگ است"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /upload [post]
func (h *UploadHandler) UploadFile(c *gin.Context) {
	fileType, ok := normalizeUploadType(c.DefaultQuery("type", "document"))
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid upload type. Allowed: document, profile"})
		return
	}

	// Get file from request
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no file provided"})
		return
	}

	// Check file size (max 10MB for images, 50MB for documents)
	maxSize := int64(50 * 1024 * 1024) // 50MB
	if fileType == "profile" {
		maxSize = int64(10 * 1024 * 1024) // 10MB
	}

	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("file too large (max %dMB)", maxSize/(1024*1024)),
		})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".xls":  true,
		".xlsx": true,
		".txt":  true,
	}

	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid file type. Allowed: jpg, jpeg, png, gif, pdf, doc, docx, xls, xlsx, txt",
		})
		return
	}

	// Generate unique filename
	newFilename := generateUniqueFilename(file.Filename)

	// Create subdirectory based on type
	subdir := filepath.Join(h.uploadPath, fileType)
	if err := os.MkdirAll(subdir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create upload directory"})
		return
	}

	// Save file
	filePath := filepath.Join(subdir, newFilename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save file"})
		return
	}

	// Return file URL
	fileURL := fmt.Sprintf("/uploads/%s/%s", fileType, newFilename)

	c.JSON(http.StatusOK, UploadResponse{
		URL:      fileURL,
		Filename: file.Filename,
		Size:     file.Size,
	})
}

// UploadMultiple godoc
// @Summary آپلود چند فایل
// @Description آپلود همزمان چندین فایل
// @Tags آپلود
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "فایل‌ها برای آپلود" multiple
// @Param type query string false "نوع فایل (profile, document)" default(document)
// @Success 200 {array} UploadResponse "فایل‌ها با موفقیت آپلود شدند"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /upload/multiple [post]
func (h *UploadHandler) UploadMultiple(c *gin.Context) {
	fileType, ok := normalizeUploadType(c.DefaultQuery("type", "document"))
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid upload type. Allowed: document, profile"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "failed to parse form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no files provided"})
		return
	}

	// Limit to 10 files per request
	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "maximum 10 files per request"})
		return
	}

	responses := make([]UploadResponse, 0, len(files))
	maxSize := int64(50 * 1024 * 1024) // 50MB

	for _, file := range files {
		if file.Size > maxSize {
			continue // Skip files that are too large
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedExts := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".pdf":  true,
			".doc":  true,
			".docx": true,
			".xls":  true,
			".xlsx": true,
			".txt":  true,
		}

		if !allowedExts[ext] {
			continue // Skip invalid file types
		}

		newFilename := generateUniqueFilename(file.Filename)

		subdir := filepath.Join(h.uploadPath, fileType)
		if err := os.MkdirAll(subdir, 0755); err != nil {
			continue
		}

		filePath := filepath.Join(subdir, newFilename)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			continue
		}

		fileURL := fmt.Sprintf("/uploads/%s/%s", fileType, newFilename)
		responses = append(responses, UploadResponse{
			URL:      fileURL,
			Filename: file.Filename,
			Size:     file.Size,
		})
	}

	if len(responses) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no valid files uploaded"})
		return
	}

	c.JSON(http.StatusOK, responses)
}
