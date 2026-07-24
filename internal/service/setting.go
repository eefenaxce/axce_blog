package service

import (
	"context"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
)

type SettingService struct {
	settingRepo repository.SettingRepository
}

func NewSettingService(settingRepo repository.SettingRepository) *SettingService {
	return &SettingService{settingRepo: settingRepo}
}

func (s *SettingService) Get(ctx context.Context, key string) (string, error) {
	setting, err := s.settingRepo.Get(ctx, key)
	if err != nil {
		return "", nil
	}
	return setting.Value, nil
}

func (s *SettingService) Set(ctx context.Context, key, value, description string) error {
	return s.settingRepo.Set(ctx, key, value, description)
}

func (s *SettingService) List(ctx context.Context) ([]*models.Setting, error) {
	return s.settingRepo.List(ctx)
}
