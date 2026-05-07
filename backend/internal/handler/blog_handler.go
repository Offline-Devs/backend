package handler

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/noshirvani-academy/backend/internal/domain"
    "gorm.io/gorm"
)

type BlogHandler struct {
    db *gorm.DB
}

type blogInput struct {
    Title     string `json:"title" binding:"required"`
    Slug      string `json:"slug"`
    Content   string `json:"content"`
    AuthorID  string `json:"author_id"`
    Published bool   `json:"published"`
}

func NewBlogHandler(db *gorm.DB) *BlogHandler {
    return &BlogHandler{db: db}
}

func (h *BlogHandler) Create(c *gin.Context) {
    var input blogInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }
    if input.Slug == "" {
        input.Slug = slugify(input.Title)
    }

    post := domain.BlogPost{
        Title:     input.Title,
        Slug:      input.Slug,
        Content:   input.Content,
        AuthorID:  input.AuthorID,
        Published: input.Published,
    }

    if err := h.db.Create(&post).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
        return
    }
    c.JSON(http.StatusOK, post)
}

func (h *BlogHandler) Update(c *gin.Context) {
    id := c.Param("id")
    var input blogInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }
    if input.Slug == "" && input.Title != "" {
        input.Slug = slugify(input.Title)
    }

    updates := map[string]interface{}{
        "title":     input.Title,
        "slug":      input.Slug,
        "content":   input.Content,
        "author_id": input.AuthorID,
        "published": input.Published,
    }

    if err := h.db.Model(&domain.BlogPost{}).Where("id = ?", id).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update post"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *BlogHandler) Publish(c *gin.Context) {
    id := c.Param("id")
    if err := h.db.Model(&domain.BlogPost{}).Where("id = ?", id).Update("published", true).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish post"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "published"})
}

func (h *BlogHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    if err := h.db.Delete(&domain.BlogPost{}, "id = ?", id).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete post"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *BlogHandler) List(c *gin.Context) {
    var posts []domain.BlogPost
    if err := h.db.Order("created_at desc").Limit(100).Find(&posts).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load posts"})
        return
    }
    c.JSON(http.StatusOK, posts)
}

func (h *BlogHandler) PublicList(c *gin.Context) {
    var posts []domain.BlogPost
    if err := h.db.Where("published = ?", true).Order("created_at desc").Limit(50).Find(&posts).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load posts"})
        return
    }
    c.JSON(http.StatusOK, posts)
}

func (h *BlogHandler) PublicGet(c *gin.Context) {
    slug := c.Param("slug")
    var post domain.BlogPost
    if err := h.db.Where("slug = ? AND published = ?", slug, true).First(&post).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
        return
    }
    c.JSON(http.StatusOK, post)
}

func slugify(input string) string {
    output := strings.ToLower(strings.TrimSpace(input))
    output = strings.ReplaceAll(output, " ", "-")
    output = strings.ReplaceAll(output, "--", "-")
    return output
}
