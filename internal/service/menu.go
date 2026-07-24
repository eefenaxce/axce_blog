package service

import (
	"context"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
)

type MenuService struct {
	menuRepo repository.MenuRepository
}

func NewMenuService(menuRepo repository.MenuRepository) *MenuService {
	return &MenuService{menuRepo: menuRepo}
}

// GetByName returns a menu with its items by menu name.
func (s *MenuService) GetByName(ctx context.Context, name string) (*models.Menu, []*models.MenuItem, error) {
	menu, err := s.menuRepo.GetByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.menuRepo.GetItems(ctx, menu.ID)
	if err != nil {
		return menu, nil, err
	}
	return menu, items, nil
}
