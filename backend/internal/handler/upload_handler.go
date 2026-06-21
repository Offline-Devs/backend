package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
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

type UploadResponse struct {
	URL      string `json:"url" description:"آدرس دسترسی به فایل"`
	Filename string `json:"filename" description:"نام فایل"`
	Size     int64  `json:"size" description:"حجم فایل (بایت)"`
}

func NewUploadHandler(uploadPath string) *UploadHandler {
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		fmt.Printf("[WARNING] Failed to create upload directory: %v\n", err)
	}
	return &UploadHandler{uploadPath: uploadPath}
}

func generateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), hex.EncodeToString(randomBytes), ext)
}

func fileTypeConfig(fileType string) (int64, map[string]bool, error) {
	switch fileType {
	case "profile":
		return 10 * 1024 * 1024, map[string]bool{".jpg": true, ".jpeg": true, ".png": true}, nil
	case "document", "note":
		return 50 * 1024 * 1024, map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".txt": true}, nil
	default:
		return 0, nil, fmt.Errorf("invalid file type")
	}
}

func validateUploadedFile(fileHeader *multipart.FileHeader, allowedExts map[string]bool, maxSize int64) error {
	if fileHeader.Size > maxSize {
		return fmt.Errorf("file too large")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExts[ext] {
		return fmt.Errorf("invalid file extension")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}
	contentType := http.DetectContentType(buffer[:n])
	if !strings.HasPrefix(contentType, "image/") && contentType != "application/pdf" && contentType != "text/plain" && contentType != "application/msword" && contentType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" && contentType != "application/vnd.ms-excel" && contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		return fmt.Errorf("invalid file content type")
	}
	return nil
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	fileType := c.DefaultQuery("type", "document")
	maxSize, allowedExts, err := fileTypeConfig(fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no file provided"})
		return
	}
	if err := validateUploadedFile(file, allowedExts, maxSize); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	newFilename := generateUniqueFilename(file.Filename)
	subdir := filepath.Join(h.uploadPath, fileType)
	if err := os.MkdirAll(subdir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create upload directory"})
		return
	}

	filePath := filepath.Join(subdir, newFilename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save file"})
		return
	}

	fileURL := fmt.Sprintf("/uploads/%s/%s", fileType, newFilename)
	c.JSON(http.StatusOK, UploadResponse{URL: fileURL, Filename: file.Filename, Size: file.Size})
}

func (h *UploadHandler) UploadMultiple(c *gin.Context) {
	fileType := c.DefaultQuery("type", "document")
	maxSize, allowedExts, err := fileTypeConfig(fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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
	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "maximum 10 files per request"})
		return
	}

	responses := make([]UploadResponse, 0, len(files))
	subdir := filepath.Join(h.uploadPath, fileType)
	if err := os.MkdirAll(subdir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create upload directory"})
		return
	}

	for _, file := range files {
		if err := validateUploadedFile(file, allowedExts, maxSize); err != nil {
			continue
		}
		newFilename := generateUniqueFilename(file.Filename)
		filePath := filepath.Join(subdir, newFilename)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			continue
		}
		responses = append(responses, UploadResponse{
			URL:      fmt.Sprintf("/uploads/%s/%s", fileType, newFilename),
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
