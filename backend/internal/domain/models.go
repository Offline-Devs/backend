package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Phone     string         `gorm:"uniqueIndex;size:20;not null" json:"phone"`
	Role      string         `gorm:"default:'student';not null" json:"role"`
	Password  string         `json:"-"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Student *Student `gorm:"foreignKey:UserID" json:"student,omitempty"`
}

type Student struct {
	ID              string                 `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID          string                 `gorm:"uniqueIndex;not null" json:"user_id"`
	FirstName       string                 `gorm:"not null" json:"first_name"`
	LastName        string                 `gorm:"not null" json:"last_name"`
	City            string                 `json:"city"`
	BirthDate       time.Time              `json:"birth_date"`
	JalaliBirthDate string                 `gorm:"size:10" json:"jalali_birth_date"`
	School          string                 `json:"school"`
	Major           string                 `json:"major"`
	ProfilePhoto    string                 `json:"profile_photo"`
	IsApproved      bool                   `gorm:"default:false" json:"is_approved"`
	ApprovalDate    *time.Time             `json:"approval_date,omitempty"`
	DynamicFields   map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"dynamic_fields"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       gorm.DeletedAt         `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type DynamicFieldDefinition struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	EntityType string         `gorm:"not null" json:"entity_type"`
	Name       string         `gorm:"not null" json:"name"`
	Label      string         `json:"label"`
	FieldType  string         `gorm:"not null" json:"field_type"`
	Options    string         `json:"options,omitempty"`
	IsRequired bool           `json:"is_required"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

type DynamicFieldValue struct {
	ID                string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	EntityType        string         `gorm:"not null;index" json:"entity_type"`
	EntityID          string         `gorm:"not null;index" json:"entity_id"`
	FieldDefinitionID string         `gorm:"not null" json:"field_definition_id"`
	Value             string         `json:"value"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

type Exam struct {
	ID            string                 `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentID     string                 `gorm:"index;not null" json:"student_id"`
	Title         string                 `json:"title"`
	ExamDate      time.Time              `json:"exam_date"`
	JalaliDate    string                 `gorm:"size:10" json:"jalali_date"`
	Major         string                 `json:"major"`
	NegativeMark  float64                `gorm:"default:0" json:"negative_mark"`
	TotalSubjects int                    `json:"total_subjects"`
	DynamicFields map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"dynamic_fields"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	DeletedAt     gorm.DeletedAt         `gorm:"index"`

	Subjects []SubjectExam `gorm:"foreignKey:ExamID" json:"subjects,omitempty"`
	Student  Student       `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

type SubjectExam struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ExamID         string `gorm:"index;not null" json:"exam_id"`
	SubjectName    string `gorm:"not null" json:"subject_name"`
	TotalQuestions int    `json:"total_questions"`
	Correct        int    `json:"correct"`
	Wrong          int    `json:"wrong"`

	Exam Exam `gorm:"foreignKey:ExamID" json:"-"`
}

type Mistake struct {
	ID             string                 `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentID      string                 `gorm:"index;not null" json:"student_id"`
	ExamID         *string                `json:"exam_id,omitempty"`
	SubjectExamID  *string                `json:"subject_exam_id,omitempty"`
	QuestionNumber int                    `json:"question_number"`
	Category       string                 `json:"category"`
	Notes          string                 `json:"notes"`
	DynamicFields  map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"dynamic_fields"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	DeletedAt      gorm.DeletedAt         `gorm:"index"`

	Student     Student     `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Exam        Exam        `gorm:"foreignKey:ExamID" json:"exam,omitempty"`
	SubjectExam SubjectExam `gorm:"foreignKey:SubjectExamID" json:"subject_exam,omitempty"`
}

type PerformanceHistory struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentID  string         `gorm:"index;not null" json:"student_id"`
	Date       time.Time      `json:"date"`
	JalaliDate string         `gorm:"size:10" json:"jalali_date"`
	Notes      string         `json:"notes"`
	Files      string         `json:"files,omitempty"`
	StudyPlan  string         `json:"study_plan,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	Student Student `gorm:"foreignKey:StudentID"`
}

type Notification struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string         `gorm:"index;not null" json:"user_id"`
	Title     string         `gorm:"not null" json:"title"`
	Body      string         `json:"body,omitempty"`
	Href      string         `json:"href,omitempty"`
	IsRead    bool           `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

type BlogPost struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Title     string         `json:"title"`
	Slug      string         `gorm:"uniqueIndex" json:"slug"`
	Content   string         `gorm:"type:text" json:"content"`
	AuthorID  string         `json:"author_id"`
	Published bool           `gorm:"default:false" json:"published"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
