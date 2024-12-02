package models

import (
	"context"

	"github.com/google/uuid"
)

type StorageInt interface {
	CreateItem(ctx context.Context, i *ItemCreateRequest) (*uuid.UUID, error)
	UpdateItem(ctx context.Context, id uuid.UUID, i *ItemUpdateRequest) error
	DeleteItem(ctx context.Context, id uuid.UUID) error
	GetItem(ctx context.Context, id uuid.UUID) (*ItemResponse, error)
	Close()
}

type AuthInt interface {
	TokenHasRole(tokenString, role string) (bool, error)
	Generate(roles []string) (string, error)
}

type NotifyInt interface {
	Send(event EventNotifications) error
	Close()
}
