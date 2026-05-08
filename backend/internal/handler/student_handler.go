package handler

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/noshirvani-academy/backend/internal/domain"
    "github.com/yourusername/noshirvani-academy/backend/pkg"
    "gorm.io/gorm"
)

type StudentHandler struct {
    db *gorm.DB
}

type studentProfileInput struct {
    FirstName        string                 `json:"first_name"`
    LastName         string                 `json:"last_name"`
    City             string                 `json:"city"`
    BirthDate        *time.Time             `json:"birth_date"`
    JalaliBirthDate  string                 `json:"jalali_birth_date"`
    School           string                 `json:"school"`
    Major            string                 `json:"major"`
    ProfilePhoto     string                 `json:"profile_photo"`
    DynamicFields    map[string]interface{} `json:"dynamic_fields"`
}

func NewStudentHandler(db *gorm.DB) *StudentHandler {
    return &StudentHandler{db: db}
}

func (h *StudentHandler) CompleteProfile(c *gin.Context) {
    var input studentProfileInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    userID, ok := c.Get("user_id")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
        return
    }

    var student domain.Student
    err := h.db.Where("user_id = ?", userID).First(&student).Error
    if err != nil && err != gorm.ErrRecordNotFound {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load profile"})
        return
    }

    if input.JalaliBirthDate != "" {
        if t, err := pkg.JalaliToGregorian(input.JalaliBirthDate); err == nil {
            input.BirthDate = &t
        }
    }

    if err == gorm.ErrRecordNotFound {
        student = domain.Student{
            UserID:          userID.(string),
            FirstName:       input.FirstName,
            LastName:        input.LastName,
            City:            input.City,
            School:          input.School,
            Major:           input.Major,
            ProfilePhoto:    input.ProfilePhoto,
            DynamicFields:   input.DynamicFields,
        }
        if input.BirthDate != nil {
            student.BirthDate = *input.BirthDate
        }
        if input.JalaliBirthDate != "" {
            student.JalaliBirthDate = input.JalaliBirthDate
        }

        if err := h.db.Create(&student).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create profile"})
            return
        }
        c.JSON(http.StatusOK, student)
        return
    }

    updates := map[string]interface{}{
        "first_name":        input.FirstName,
        "last_name":         input.LastName,
        "city":              input.City,
        "school":            input.School,
        "major":             input.Major,
        "profile_photo":     input.ProfilePhoto,
        "dynamic_fields":    input.DynamicFields,
        "jalali_birth_date": input.JalaliBirthDate,
    }
    if input.BirthDate != nil {
        updates["birth_date"] = *input.BirthDate
    }

    if err := h.db.Model(&student).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
        return
    }
    if err := h.db.Preload("User").First(&student, "id = ?", student.ID).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reload profile"})
        return
    }

    c.JSON(http.StatusOK, student)
}

func (h *StudentHandler) GetProfile(c *gin.Context) {
    userID, ok := c.Get("user_id")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
        return
    }

    var student domain.Student
    if err := h.db.Preload("User").Where("user_id = ?", userID).First(&student).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
        return
    }

    c.JSON(http.StatusOK, student)
}
