package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/pkg"
	"gorm.io/gorm"
)

var dynamicFieldNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var allowedDynamicFieldEntities = map[string]bool{
	"student": true,
	"exam":    true,
	"mistake": true,
}

var allowedDynamicFieldTypes = map[string]bool{
	"text":     true,
	"number":   true,
	"select":   true,
	"checkbox": true,
	"date":     true,
}

func normalizeDynamicFieldDefinition(input CreateDynamicFieldInput) (domain.DynamicFieldDefinition, error) {
	entityType := strings.TrimSpace(strings.ToLower(input.EntityType))
	name := strings.TrimSpace(strings.ToLower(input.Name))
	label := strings.TrimSpace(input.Label)
	fieldType := strings.TrimSpace(strings.ToLower(input.FieldType))
	options := strings.TrimSpace(input.Options)

	if !allowedDynamicFieldEntities[entityType] {
		return domain.DynamicFieldDefinition{}, errors.New("invalid dynamic field entity_type")
	}
	if !dynamicFieldNamePattern.MatchString(name) {
		return domain.DynamicFieldDefinition{}, errors.New("invalid dynamic field name")
	}
	if label == "" {
		return domain.DynamicFieldDefinition{}, errors.New("dynamic field label is required")
	}
	if !allowedDynamicFieldTypes[fieldType] {
		return domain.DynamicFieldDefinition{}, errors.New("invalid dynamic field type")
	}

	if fieldType == "select" {
		normalizedOptions, err := normalizeDynamicFieldOptions(options)
		if err != nil {
			return domain.DynamicFieldDefinition{}, err
		}
		options = normalizedOptions
	} else {
		options = ""
	}

	return domain.DynamicFieldDefinition{
		EntityType: entityType,
		Name:       name,
		Label:      label,
		FieldType:  fieldType,
		Options:    options,
		IsRequired: input.IsRequired,
	}, nil
}

func normalizeDynamicFieldOptions(raw string) (string, error) {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return "", errors.New("invalid dynamic field options")
	}
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "", errors.New("invalid dynamic field options")
		}
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return "", errors.New("invalid dynamic field options")
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (h *AdminHandler) ensureDynamicFieldNameAvailable(id string, field domain.DynamicFieldDefinition) error {
	var existing domain.DynamicFieldDefinition
	query := h.db.Where("entity_type = ? AND name = ?", field.EntityType, field.Name)
	if id != "" {
		query = query.Where("id <> ?", id)
	}
	if err := query.First(&existing).Error; err == nil {
		return errors.New("dynamic field already exists")
	} else if err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func loadActiveDynamicFields(db *gorm.DB, entityType string) ([]domain.DynamicFieldDefinition, error) {
	var fields []domain.DynamicFieldDefinition
	err := db.Where("entity_type = ? AND is_active = true", entityType).
		Order("created_at asc").
		Find(&fields).Error
	return fields, err
}

func validateAndCleanDynamicValues(db *gorm.DB, entityType string, values map[string]interface{}) (map[string]interface{}, error) {
	if values == nil {
		values = map[string]interface{}{}
	}
	fields, err := loadActiveDynamicFields(db, entityType)
	if err != nil {
		return nil, err
	}

	cleaned := map[string]interface{}{}
	for _, field := range fields {
		value, exists := values[field.Name]
		if isEmptyDynamicValue(value) {
			if field.IsRequired {
				return nil, fmt.Errorf("%s is required", field.Name)
			}
			continue
		}
		normalized, err := normalizeDynamicValue(field, value)
		if err != nil {
			return nil, err
		}
		if exists {
			cleaned[field.Name] = normalized
		}
	}
	return cleaned, nil
}

func isEmptyDynamicValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case bool:
		return !typed
	case float64:
		return false
	case int:
		return false
	default:
		return false
	}
}

func normalizeDynamicValue(field domain.DynamicFieldDefinition, value interface{}) (interface{}, error) {
	switch field.FieldType {
	case "text":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be text", field.Name)
		}
		return strings.TrimSpace(text), nil
	case "number":
		switch typed := value.(type) {
		case float64:
			return typed, nil
		case int:
			return typed, nil
		default:
			return nil, fmt.Errorf("%s must be number", field.Name)
		}
	case "checkbox":
		checked, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s must be boolean", field.Name)
		}
		return checked, nil
	case "date":
		date, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be date", field.Name)
		}
		canonical, err := pkg.CanonicalJalaliDate(date)
		if err != nil {
			return nil, fmt.Errorf("%s must be date", field.Name)
		}
		return canonical, nil
	case "select":
		selected, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be select option", field.Name)
		}
		var options []string
		if err := json.Unmarshal([]byte(field.Options), &options); err != nil {
			return nil, fmt.Errorf("%s has invalid options", field.Name)
		}
		for _, option := range options {
			if selected == option {
				return selected, nil
			}
		}
		return nil, fmt.Errorf("%s must be select option", field.Name)
	default:
		return nil, fmt.Errorf("%s has invalid type", field.Name)
	}
}
