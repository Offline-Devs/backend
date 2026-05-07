package handler

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/noshirvani-academy/backend/internal/domain"
    "gorm.io/gorm"
)

type AdminHandler struct {
    db *gorm.DB
}

type createDynamicFieldInput struct {
    EntityType string `json:"entity_type" binding:"required"`
    Name       string `json:"name" binding:"required"`
    Label      string `json:"label"`
    FieldType  string `json:"field_type" binding:"required"`
    Options    string `json:"options"`
    IsRequired bool   `json:"is_required"`
}

type updateStudentInput struct {
    FirstName    *string `json:"first_name"`
    LastName     *string `json:"last_name"`
    City         *string `json:"city"`
    School       *string `json:"school"`
    Major        *string `json:"major"`
    IsApproved   *bool   `json:"is_approved"`
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
    return &AdminHandler{db: db}
}

func (h *AdminHandler) ListStudents(c *gin.Context) {
    var students []domain.Student
    if err := h.db.Preload("User").Limit(100).Order("created_at desc").Find(&students).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load students"})
        return
    }
    c.JSON(http.StatusOK, students)
}

func (h *AdminHandler) GetStudent(c *gin.Context) {
    id := c.Param("id")
    var student domain.Student
    if err := h.db.Preload("User").First(&student, "id = ?", id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
        return
    }
    c.JSON(http.StatusOK, student)
}

func (h *AdminHandler) UpdateStudent(c *gin.Context) {
    id := c.Param("id")
    var input updateStudentInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    updates := map[string]interface{}{}
    if input.FirstName != nil {
        updates["first_name"] = *input.FirstName
    }
    if input.LastName != nil {
        updates["last_name"] = *input.LastName
    }
    if input.City != nil {
        updates["city"] = *input.City
    }
    if input.School != nil {
        updates["school"] = *input.School
    }
    if input.Major != nil {
        updates["major"] = *input.Major
    }
    if input.IsApproved != nil {
        updates["is_approved"] = *input.IsApproved
        if *input.IsApproved {
            now := time.Now()
            updates["approval_date"] = &now
        } else {
            updates["approval_date"] = nil
        }
    }

    if err := h.db.Model(&domain.Student{}).Where("id = ?", id).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update student"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *AdminHandler) DeleteStudent(c *gin.Context) {
    id := c.Param("id")
    if err := h.db.Delete(&domain.Student{}, "id = ?", id).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete student"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *AdminHandler) ApproveStudent(c *gin.Context) {
    id := c.Param("id")
    now := time.Now()
    if err := h.db.Model(&domain.Student{}).Where("id = ?", id).Updates(map[string]interface{}{
        "is_approved":   true,
        "approval_date": &now,
    }).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve student"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

func (h *AdminHandler) GetDynamicFields(c *gin.Context) {
    entityType := c.Query("entity_type")
    q := h.db.Model(&domain.DynamicFieldDefinition{})
    if entityType != "" {
        q = q.Where("entity_type = ?", entityType)
    }

    var fields []domain.DynamicFieldDefinition
    if err := q.Order("created_at desc").Find(&fields).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load fields"})
        return
    }
    c.JSON(http.StatusOK, fields)
}

func (h *AdminHandler) CreateDynamicField(c *gin.Context) {
    var input createDynamicFieldInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    field := domain.DynamicFieldDefinition{
        EntityType: input.EntityType,
        Name:       input.Name,
        Label:      input.Label,
        FieldType:  input.FieldType,
        Options:    input.Options,
        IsRequired: input.IsRequired,
    }

    if err := h.db.Create(&field).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create field"})
        return
    }

    c.JSON(http.StatusOK, field)
}

func (h *AdminHandler) UpdateDynamicField(c *gin.Context) {
    id := c.Param("id")
    var input createDynamicFieldInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    updates := map[string]interface{}{
        "entity_type": input.EntityType,
        "name":        input.Name,
        "label":       input.Label,
        "field_type":  input.FieldType,
        "options":     input.Options,
        "is_required": input.IsRequired,
    }

    if err := h.db.Model(&domain.DynamicFieldDefinition{}).Where("id = ?", id).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update field"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *AdminHandler) DeleteDynamicField(c *gin.Context) {
    id := c.Param("id")
    if err := h.db.Delete(&domain.DynamicFieldDefinition{}, "id = ?", id).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete field"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
