package domain

import (
	"time"

	"gorm.io/gorm"
)

const (
	StudentProfileStatusPending  = "pending"
	StudentProfileStatusApproved = "approved"
	StudentProfileStatusRejected = "rejected"
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

	StudentProfile *StudentProfile `gorm:"foreignKey:UserID" json:"student_profile,omitempty"`
}

type StudentProfile struct {
	ID              string                 `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID          string                 `gorm:"uniqueIndex;not null" json:"user_id"`
	FirstName       string                 `gorm:"not null" json:"first_name"`
	LastName        string                 `gorm:"not null" json:"last_name"`
	City            string                 `gorm:"not null" json:"city"`
	School          string                 `gorm:"not null" json:"school"`
	Major           string                 `gorm:"not null" json:"major"`
	BirthDate       time.Time              `json:"birth_date"`
	JalaliBirthDate string                 `gorm:"size:10;not null" json:"jalali_birth_date"`
	ProfilePhoto    string                 `json:"profile_photo"`
	Status          string                 `gorm:"size:20;not null;default:'pending';index" json:"status"`
	IsApproved      bool                   `gorm:"default:false" json:"is_approved"`
	ApprovalDate    *time.Time             `json:"approval_date,omitempty"`
	ReviewedAt      *time.Time             `json:"reviewed_at,omitempty"`
	ReviewedBy      *string                `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	RejectionReason string                 `json:"rejection_reason,omitempty"`
	LastSubmittedAt *time.Time             `json:"last_submitted_at,omitempty"`
	DynamicFields   map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"dynamic_fields"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       gorm.DeletedAt         `gorm:"index" json:"-"`
	User            User                   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Exams           []Exam                 `gorm:"foreignKey:StudentProfileID" json:"exams,omitempty"`
	MistakeAnalyses []MistakeAnalysis      `gorm:"foreignKey:StudentProfileID" json:"mistake_analyses,omitempty"`
	StudyPlans      []StudyPlan            `gorm:"foreignKey:StudentProfileID" json:"study_plans,omitempty"`
	AdminNotes      []AdminNote            `gorm:"foreignKey:StudentProfileID" json:"admin_notes,omitempty"`
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
	ID               string                 `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentProfileID string                 `gorm:"column:student_id;index;not null" json:"student_profile_id"`
	Title            string                 `gorm:"not null" json:"title"`
	Date             time.Time              `gorm:"column:exam_date;not null;index" json:"date"`
	JalaliDate       string                 `gorm:"size:10;not null;index" json:"jalali_date"`
	Major            string                 `gorm:"not null" json:"major"`
	TotalSubjects    int                    `json:"total_subjects"`
	DynamicFields    map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"dynamic_fields"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DeletedAt        gorm.DeletedAt         `gorm:"index"`

	Subjects       []ExamSubject  `gorm:"foreignKey:ExamID" json:"subjects,omitempty"`
	StudentProfile StudentProfile `gorm:"foreignKey:StudentProfileID" json:"student_profile,omitempty"`
}

type ExamSubject struct {
	ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ExamID         string    `gorm:"index;not null" json:"exam_id"`
	SubjectName    string    `gorm:"not null" json:"subject_name"`
	TotalQuestions int       `json:"total_questions"`
	Answered       int       `json:"answered"`
	Correct        int       `json:"correct"`
	Wrong          int       `json:"wrong"`
	Blank          int       `json:"blank"`
	Percentage     float64   `gorm:"type:numeric(5,2);default:0" json:"percentage"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Exam Exam `gorm:"foreignKey:ExamID" json:"-"`
}

type MistakeAnalysis struct {
	ID               string                 `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentProfileID string                 `gorm:"column:student_id;index;not null" json:"student_profile_id"`
	ExamID           *string                `json:"exam_id,omitempty"`
	ExamSubjectID    *string                `gorm:"column:subject_exam_id" json:"exam_subject_id,omitempty"`
	QuestionNumber   int                    `json:"question_number"`
	Category         string                 `gorm:"not null" json:"category"`
	Notes            string                 `json:"notes"`
	DynamicFields    map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"dynamic_fields"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DeletedAt        gorm.DeletedAt         `gorm:"index"`

	StudentProfile StudentProfile `gorm:"foreignKey:StudentProfileID" json:"student_profile,omitempty"`
	Exam           Exam           `gorm:"foreignKey:ExamID" json:"exam,omitempty"`
	ExamSubject    ExamSubject    `gorm:"foreignKey:ExamSubjectID" json:"exam_subject,omitempty"`
}

type StudyPlan struct {
	ID               string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentProfileID string         `gorm:"column:student_id;index;not null" json:"student_profile_id"`
	AssignedBy       string         `gorm:"type:uuid;not null" json:"assigned_by"`
	Title            string         `gorm:"not null" json:"title"`
	Description      string         `gorm:"type:text" json:"description"`
	StartDate        time.Time      `gorm:"not null;index" json:"start_date"`
	EndDate          time.Time      `gorm:"not null;index" json:"end_date"`
	JalaliStartDate  string         `gorm:"size:10;not null" json:"jalali_start_date"`
	JalaliEndDate    string         `gorm:"size:10;not null" json:"jalali_end_date"`
	Attachments      string         `gorm:"type:text" json:"attachments"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`

	StudentProfile StudentProfile `gorm:"foreignKey:StudentProfileID" json:"student_profile,omitempty"`
}

type AdminNote struct {
	ID               string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	StudentProfileID string         `gorm:"column:student_id;index;not null" json:"student_profile_id"`
	AdminID          string         `gorm:"type:uuid;not null;index" json:"admin_id"`
	Title            string         `gorm:"not null" json:"title"`
	Body             string         `gorm:"type:text;not null" json:"body"`
	NoteDate         time.Time      `gorm:"column:date;not null;index" json:"date"`
	JalaliDate       string         `gorm:"size:10;not null" json:"jalali_date"`
	AttachmentURL    string         `json:"attachment_url,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`

	StudentProfile StudentProfile `gorm:"foreignKey:StudentProfileID" json:"student_profile,omitempty"`
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

type Student = StudentProfile

type SubjectExam = ExamSubject

type Mistake = MistakeAnalysis
