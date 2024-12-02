package models

import (
	"errors"

	"github.com/google/uuid"
)

type ItemCreateRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	EmployeeCount int    `json:"employee_count"`
	IsRegistered  bool   `json:"is_registered"`
	Type          string `json:"type"`
}

type ItemUpdateRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	EmployeeCount *int    `json:"employee_count"`
	IsRegistered  *bool   `json:"is_registered"`
	Type          *string `json:"type"`
}

type ItemResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	EmployeeCount int       `json:"employee_count"`
	IsRegistered  bool      `json:"is_registered"`
	Type          string    `json:"type"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type EventNotifications struct {
	ID        uuid.UUID `json:"id"`
	Event     string    `json:"event"`
	Timestamp int64     `json:"timestamp"`
}

var AcceptableLegalTypes = map[string]struct{}{
	"Corporations":        {},
	"NonProfit":           {},
	"Cooperative":         {},
	"Sole Proprietorship": {},
}

var (
	ErrNotFound           = errors.New("Item not found")
	ErrNothingToDo        = errors.New("Empty request - nothing to do")
	ErrDuplicateName      = errors.New("Duplicate item name")
	ErrInvalidID          = errors.New("Invalid id")
	ErrInvalidName        = errors.New("Invalid name")
	ErrInvalidDescription = errors.New("Invalid description")
	ErrInvalidType        = errors.New("Invalid type")
	ErrInvalidRequest     = errors.New("Invalid request")
	ErrDBError            = errors.New("DB error")
	ErrJWTInvalid         = errors.New("Invalid JWT")
	ErrJWTRoleMissing     = errors.New("Access denied")
	ErrJWTInvalidMethod   = errors.New("Invalid signing method")
)

const (
	RoleReader = "reader"
	RoleWriter = "writer"

	EventTypeCreated = "created"
	EventTypeUpdated = "updated"
	EventTypeDeleted = "deleted"
)
