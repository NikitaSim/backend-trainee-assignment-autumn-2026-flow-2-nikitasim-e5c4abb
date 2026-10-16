package service

import (
	"context"
	"errors"

	"avito-kitchen/internal/model"
	"github.com/google/uuid"
)

type CartRepository interface {
	Get(context.Context, uuid.UUID) (model.Cart, error)
	AddItem(context.Context, uuid.UUID, uuid.UUID, int) error
	UpdateItem(context.Context, uuid.UUID, uuid.UUID, int) error
	DeleteItem(context.Context, uuid.UUID, uuid.UUID) error
	Clear(context.Context, uuid.UUID) error
}

type CartService struct {
	repo CartRepository
	menu MenuRepository
}

func NewCartService(r CartRepository, m MenuRepository) *CartService {
	return &CartService{repo: r, menu: m}
}

func (s *CartService) Get(ctx context.Context, session uuid.UUID) (model.Cart, error) {
	return s.repo.Get(ctx, session)
}

func (s *CartService) Add(ctx context.Context, session, menuItem uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	return s.repo.AddItem(ctx, session, menuItem, quantity)
}

func (s *CartService) Update(ctx context.Context, session, item uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	return s.repo.UpdateItem(ctx, session, item, quantity)
}

func (s *CartService) Delete(ctx context.Context, session, item uuid.UUID) error {
	return s.repo.DeleteItem(ctx, session, item)
}
