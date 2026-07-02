package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type NotificationHandler struct {
	db *gorm.DB
}

type NotificationsResponse struct {
	Notifications []domain.Notification `json:"notifications"`
	UnreadCount   int64                 `json:"unread_count"`
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

func (h *NotificationHandler) ListStudentNotifications(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var notifications []domain.Notification
	if err := h.db.Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load notifications"})
		return
	}

	var unreadCount int64
	if err := h.db.Model(&domain.Notification{}).Where("user_id = ? AND is_read = false", userID).Count(&unreadCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load notifications"})
		return
	}

	c.JSON(http.StatusOK, NotificationsResponse{Notifications: notifications, UnreadCount: unreadCount})
}

func (h *NotificationHandler) MarkNotificationRead(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	result := h.db.Model(&domain.Notification{}).
		Where("id = ? AND user_id = ?", c.Param("id"), userID).
		Update("is_read", true)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update notification"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "notification not found"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
