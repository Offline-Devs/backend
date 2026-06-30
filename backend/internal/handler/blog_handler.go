package handler

import (
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type BlogHandler struct {
	db *gorm.DB
}

// BlogInput داده‌های ورودی برای نوشته بلاگ
type BlogInput struct {
	Title     string `json:"title" binding:"required" description:"عنوان نوشته"`
	Slug      string `json:"slug" description:"نشانی URL نوشته (به صورت خودکار ایجاد می‌شود اگر خالی باشد)"`
	Content   string `json:"content" description:"محتوای نوشته"`
	AuthorID  string `json:"author_id" description:"شناسه نویسنده"`
	Published bool   `json:"published" example:"false" description:"وضعیت انتشار نوشته"`
}

// Deprecated: استفاده از BlogInput کنید
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

// Create godoc
// @Summary ایجاد نوشته بلاگ جدید
// @Description یک نوشته بلاگ جدید را ایجاد می‌کند (فقط برای مدیران)
// @Tags بلاگ
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body BlogInput true "اطلاعات نوشته"
// @Success 201 {object} domain.BlogPost "نوشته با موفقیت ایجاد شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog [post]
func (h *BlogHandler) Create(c *gin.Context) {
	var input BlogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	input.Slug = slugify(firstNonEmpty(input.Slug, input.Title))

	post := domain.BlogPost{
		Title:     input.Title,
		Slug:      input.Slug,
		Content:   input.Content,
		AuthorID:  input.AuthorID,
		Published: input.Published,
	}

	if err := h.db.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create post"})
		return
	}
	c.JSON(http.StatusOK, post)
}

// Update godoc
// @Summary بروزرسانی نوشته بلاگ
// @Description اطلاعات یک نوشته بلاگ را بروزرسانی می‌کند (فقط برای مدیران)
// @Tags بلاگ
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "شناسه نوشته"
// @Param input body BlogInput true "اطلاعات جدید نوشته"
// @Success 200 {object} map[string]string "نوشته با موفقیت بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog/{id} [put]
func (h *BlogHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var input BlogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	input.Slug = slugify(firstNonEmpty(input.Slug, input.Title))

	updates := map[string]interface{}{
		"title":     input.Title,
		"slug":      input.Slug,
		"content":   input.Content,
		"author_id": input.AuthorID,
		"published": input.Published,
	}

	result := h.db.Model(&domain.BlogPost{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update post"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "post not found"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// Publish godoc
// @Summary انتشار نوشته بلاگ
// @Description یک نوشته بلاگ را برای عموم قابل دسترس می‌کند (فقط برای مدیران)
// @Tags بلاگ
// @Security BearerAuth
// @Param id path string true "شناسه نوشته"
// @Success 200 {object} map[string]string "نوشته با موفقیت منتشر شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog/{id}/publish [put]
func (h *BlogHandler) Publish(c *gin.Context) {
	id := c.Param("id")
	result := h.db.Model(&domain.BlogPost{}).Where("id = ?", id).Update("published", true)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to publish post"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "post not found"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "published"})
}

// Delete godoc
// @Summary حذف نوشته بلاگ
// @Description یک نوشته بلاگ را حذف می‌کند (فقط برای مدیران)
// @Tags بلاگ
// @Security BearerAuth
// @Param id path string true "شناسه نوشته"
// @Success 200 {object} map[string]string "نوشته با موفقیت حذف شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog/{id} [delete]
func (h *BlogHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	result := h.db.Delete(&domain.BlogPost{}, "id = ?", id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete post"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "post not found"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// List godoc
// @Summary دریافت لیست تمام نوشته‌های بلاگ
// @Description تمام نوشته‌های بلاگ (منتشر و منتشرنشده) را دریافت می‌کند (فقط برای مدیران)
// @Tags بلاگ
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.BlogPost "لیست نوشته‌ها"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog [get]
func (h *BlogHandler) List(c *gin.Context) {
	var posts []domain.BlogPost
	if err := h.db.Order("created_at desc").Limit(100).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load posts"})
		return
	}
	c.JSON(http.StatusOK, posts)
}

// PublicList godoc
// @Summary دریافت لیست نوشته‌های منتشر شده
// @Description تمام نوشته‌های منتشر‌شده را دریافت می‌کند (بدون نیاز به احراز هویت)
// @Tags بلاگ
// @Produce json
// @Success 200 {array} domain.BlogPost "لیست نوشته‌های منتشر‌شده"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog [get]
func (h *BlogHandler) PublicList(c *gin.Context) {
	var posts []domain.BlogPost
	if err := h.db.Where("published = ?", true).Order("created_at desc").Limit(50).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load posts"})
		return
	}
	c.JSON(http.StatusOK, posts)
}

// PublicGet godoc
// @Summary دریافت نوشته بلاگ با slug
// @Description یک نوشته منتشر شده را با استفاده از slug آن دریافت می‌کند (بدون نیاز به احراز هویت)
// @Tags بلاگ
// @Produce json
// @Param slug path string true "نشانی URL نوشته"
// @Success 200 {object} domain.BlogPost "نوشته بلاگ"
// @Failure 404 {object} ErrorResponse "نوشته یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /blog/{slug} [get]
func (h *BlogHandler) PublicGet(c *gin.Context) {
	slug := strings.Trim(c.Param("slug"), "/")
	if decoded, err := url.PathUnescape(slug); err == nil {
		slug = strings.Trim(decoded, "/")
	}
	if slug == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "post not found"})
		return
	}
	var post domain.BlogPost
	if err := h.db.Where("slug = ? AND published = ?", slug, true).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "post not found"})
		return
	}
	c.JSON(http.StatusOK, post)
}

func slugify(input string) string {
	input = strings.TrimSpace(input)
	var builder strings.Builder
	lastHyphen := false

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
			lastHyphen = false
			continue
		}
		if !lastHyphen && builder.Len() > 0 {
			builder.WriteRune('-')
			lastHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
