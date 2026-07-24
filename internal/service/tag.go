package service

import (
	"context"
	"errors"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

var (
	ErrTagNotFound = errors.New("tag not found")
)

type TagService struct {
	tagRepo        repository.TagRepository
	articleTagRepo repository.ArticleTagRepository
	redisClient    *utils.RedisClient
}

func NewTagService(tagRepo repository.TagRepository, articleTagRepo repository.ArticleTagRepository, redisClient *utils.RedisClient) *TagService {
	return &TagService{
		tagRepo:        tagRepo,
		articleTagRepo: articleTagRepo,
		redisClient:    redisClient,
	}
}

type CreateTagInput struct {
	Name string `validate:"required,max=50"`
	Icon string
}

func (s *TagService) Create(ctx context.Context, input CreateTagInput) (*models.Tag, error) {
	tag := &models.Tag{
		Name: input.Name,
		Slug: utils.GenerateSlug(input.Name),
		Icon: input.Icon,
	}

	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.Delete(ctx, "tags")
	}

	return tag, nil
}

func (s *TagService) GetByID(ctx context.Context, id int) (*models.Tag, error) {
	tag, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTagNotFound
	}
	return tag, nil
}

func (s *TagService) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	tag, err := s.tagRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, ErrTagNotFound
	}
	return tag, nil
}

type UpdateTagInput struct {
	Name string `validate:"required,max=50"`
	Icon string
}

func (s *TagService) Update(ctx context.Context, id int, input UpdateTagInput) (*models.Tag, error) {
	tag, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTagNotFound
	}

	tag.Name = input.Name
	tag.Slug = utils.GenerateSlug(input.Name)
	tag.Icon = input.Icon

	if err := s.tagRepo.Update(ctx, tag); err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.Delete(ctx, "tags")
	}

	return tag, nil
}

func (s *TagService) Delete(ctx context.Context, id int) error {
	return s.tagRepo.Delete(ctx, id)
}

func (s *TagService) List(ctx context.Context) ([]*models.Tag, error) {
	return s.tagRepo.List(ctx)
}

// GetTagPostCounts returns the number of non-deleted articles for each tag.
func (s *TagService) GetTagPostCounts(ctx context.Context) (map[int]int, error) {
	return s.articleTagRepo.CountAll(ctx)
}
