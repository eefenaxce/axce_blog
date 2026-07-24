package service

import (
	"context"
	"errors"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

var (
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategoryHasArticles = errors.New("category has articles")
)

type CategoryService struct {
	categoryRepo repository.CategoryRepository
	articleRepo  repository.ArticleRepository
	redisClient  *utils.RedisClient
}

func NewCategoryService(categoryRepo repository.CategoryRepository, articleRepo repository.ArticleRepository, redisClient *utils.RedisClient) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		articleRepo:  articleRepo,
		redisClient:  redisClient,
	}
}

type CreateCategoryInput struct {
	Name        string `validate:"required,max=100"`
	Slug        string
	Description string
	Icon        string
}

func (s *CategoryService) Create(ctx context.Context, input CreateCategoryInput) (*models.Category, error) {
	category := &models.Category{
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Icon:        input.Icon,
	}

	if category.Slug == "" {
		category.Slug = utils.GenerateSlug(input.Name)
	}

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.Delete(ctx, "categories")
	}

	return category, nil
}

func (s *CategoryService) GetByID(ctx context.Context, id int) (*models.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

func (s *CategoryService) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	category, err := s.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

type UpdateCategoryInput struct {
	Name        string `validate:"required,max=100"`
	Slug        string
	Description string
	Icon        string
}

func (s *CategoryService) Update(ctx context.Context, id int, input UpdateCategoryInput) (*models.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	category.Name = input.Name

	if input.Slug != "" {
		category.Slug = input.Slug
	} else {
		category.Slug = utils.GenerateSlug(input.Name)
	}

	category.Description = input.Description
	category.Icon = input.Icon

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	if s.redisClient != nil {
		s.redisClient.Delete(ctx, "categories")
	}

	return category, nil
}

func (s *CategoryService) Delete(ctx context.Context, id int) (int, error) {
	count, err := s.categoryRepo.GetArticleCount(ctx, id)
	if err != nil {
		return 0, err
	}

	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		return 0, err
	}

	if s.redisClient != nil {
		s.redisClient.Delete(ctx, "categories")
	}

	return count, nil
}

type CategoryWithCount struct {
	*models.Category
	ArticleCount int `json:"article_count"`
}

func (s *CategoryService) List(ctx context.Context) ([]*CategoryWithCount, error) {
	categories, err := s.categoryRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []*CategoryWithCount
	for _, cat := range categories {
		count, _ := s.categoryRepo.GetArticleCount(ctx, cat.ID)
		result = append(result, &CategoryWithCount{
			Category:     cat,
			ArticleCount: count,
		})
	}

	return result, nil
}
