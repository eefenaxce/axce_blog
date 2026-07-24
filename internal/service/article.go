package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

var htmlTagRe = regexp.MustCompile("<[^>]*>")

// generateSummary strips HTML tags and truncates content to a plain-text summary.
func generateSummary(content string, maxLen int) string {
	text := htmlTagRe.ReplaceAllString(content, " ")
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen]
}

var (
	ErrArticleNotFound = errors.New("article not found")
	ErrSlugExists      = errors.New("slug already exists")
)

type ArticleService struct {
	articleRepo         repository.ArticleRepository
	upvoteRepo          repository.ArticleUpvoteRepository
	tagRepo             repository.TagRepository
	articleTagRepo      repository.ArticleTagRepository
	articleCategoryRepo repository.ArticleCategoryRepository
	categoryRepo        repository.CategoryRepository
	redisClient         *utils.RedisClient
}

func NewArticleService(
	articleRepo repository.ArticleRepository,
	upvoteRepo repository.ArticleUpvoteRepository,
	tagRepo repository.TagRepository,
	articleTagRepo repository.ArticleTagRepository,
	articleCategoryRepo repository.ArticleCategoryRepository,
	categoryRepo repository.CategoryRepository,
	redisClient *utils.RedisClient,
) *ArticleService {
	return &ArticleService{
		articleRepo:         articleRepo,
		upvoteRepo:          upvoteRepo,
		tagRepo:             tagRepo,
		articleTagRepo:      articleTagRepo,
		articleCategoryRepo: articleCategoryRepo,
		categoryRepo:        categoryRepo,
		redisClient:         redisClient,
	}
}

type CreateArticleInput struct {
	Title          string `validate:"required,max=255"`
	Summary        string
	Content        string `validate:"required"`
	CoverURL       string
	CategoryIDs    []int
	TagIDs         []int
	Status         string
	CommentEnabled *bool
	UserID         int
}

func (s *ArticleService) Create(ctx context.Context, input CreateArticleInput) (*models.Article, error) {
	slug := utils.GenerateSlug(input.Title)

	existing, _ := s.articleRepo.GetBySlug(ctx, slug)
	if existing != nil {
		slug = slug + "-" + time.Now().Format("20060102150405")
	}

	summary := input.Summary
	if summary == "" {
		summary = generateSummary(input.Content, 200)
	}

	commentEnabled := true
	if input.CommentEnabled != nil {
		commentEnabled = *input.CommentEnabled
	}

	article := &models.Article{
		Title:          input.Title,
		Slug:           slug,
		Summary:        summary,
		Content:        input.Content,
		CoverURL:       input.CoverURL,
		Status:         input.Status,
		CommentEnabled: commentEnabled,
		UserID:         input.UserID,
	}

	if err := s.articleRepo.Create(ctx, article); err != nil {
		return nil, err
	}

	for _, categoryID := range input.CategoryIDs {
		s.articleCategoryRepo.Create(ctx, article.ID, categoryID)
	}

	for _, tagID := range input.TagIDs {
		s.articleTagRepo.Create(ctx, article.ID, tagID)
	}

	return article, nil
}

func (s *ArticleService) GetByID(ctx context.Context, id int) (*models.Article, error) {
	if s.redisClient != nil {
		cached, _ := s.redisClient.Get(ctx, "article:"+string(rune(id)))
		if cached != "" {
			// In production, deserialize cached data
		}
	}

	article, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArticleNotFound
	}

	categories, err := s.articleCategoryRepo.GetByArticle(ctx, id)
	if err == nil {
		article.Categories = categories
	}

	tags, err := s.articleTagRepo.GetByArticle(ctx, id)
	if err == nil {
		article.Tags = tags
	}

	s.fillUpvoteCounts(ctx, []*models.Article{article})

	return article, nil
}

func (s *ArticleService) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	article, err := s.articleRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, ErrArticleNotFound
	}

	categories, err := s.articleCategoryRepo.GetByArticle(ctx, article.ID)
	if err == nil {
		article.Categories = categories
	}

	tags, err := s.articleTagRepo.GetByArticle(ctx, article.ID)
	if err == nil {
		article.Tags = tags
	}

	s.fillUpvoteCounts(ctx, []*models.Article{article})

	return article, nil
}

func (s *ArticleService) IncrementViewCount(ctx context.Context, id int) error {
	return s.articleRepo.IncrementViewCount(ctx, id)
}

func (s *ArticleService) fillUpvoteCounts(ctx context.Context, articles []*models.Article) {
	if len(articles) == 0 {
		return
	}
	ids := make([]int, len(articles))
	for i, a := range articles {
		ids[i] = a.ID
	}
	counts, _ := s.upvoteRepo.CountByArticles(ctx, ids)
	for _, a := range articles {
		a.UpvoteCount = counts[a.ID]
	}
}

func (s *ArticleService) GetUpvoteCount(ctx context.Context, articleID int) (int, error) {
	return s.upvoteRepo.Count(ctx, articleID)
}

func (s *ArticleService) GetUpvoteCounts(ctx context.Context, articleIDs []int) (map[int]int, error) {
	return s.upvoteRepo.CountByArticles(ctx, articleIDs)
}

func (s *ArticleService) ToggleUpvote(ctx context.Context, articleID int, userID int, ip string) (int, error) {
	return s.upvoteRepo.Toggle(ctx, articleID, userID, ip)
}

type UpdateArticleInput struct {
	Title          string `validate:"required,max=255"`
	Summary        string
	Content        string `validate:"required"`
	CoverURL       string
	CategoryIDs    []int
	TagIDs         []int
	Status         string
	CommentEnabled *bool
}

func (s *ArticleService) Update(ctx context.Context, id int, input UpdateArticleInput) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArticleNotFound
	}

	article.Title = input.Title
	article.Content = input.Content
	article.CoverURL = input.CoverURL
	article.Status = input.Status
	if input.CommentEnabled != nil {
		article.CommentEnabled = *input.CommentEnabled
	}

	if input.Summary != "" {
		article.Summary = input.Summary
	} else {
		article.Summary = generateSummary(input.Content, 200)
	}

	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, err
	}

	if input.CategoryIDs != nil {
		s.articleCategoryRepo.DeleteByArticle(ctx, id)
		for _, categoryID := range input.CategoryIDs {
			s.articleCategoryRepo.Create(ctx, id, categoryID)
		}
	}

	if input.TagIDs != nil {
		s.articleTagRepo.DeleteByArticle(ctx, id)
		for _, tagID := range input.TagIDs {
			s.articleTagRepo.Create(ctx, id, tagID)
		}
	}

	if s.redisClient != nil {
		s.redisClient.Delete(ctx, "article:"+string(rune(id)))
	}

	return article, nil
}

func (s *ArticleService) Delete(ctx context.Context, id int) error {
	return s.articleRepo.Delete(ctx, id)
}

func (s *ArticleService) List(ctx context.Context, offset, limit int) ([]*models.Article, int, error) {
	articles, total, err := s.articleRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleCategories(ctx, articles); err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleTags(ctx, articles); err != nil {
		return nil, 0, err
	}
	s.fillUpvoteCounts(ctx, articles)
	return articles, total, nil
}

func (s *ArticleService) ListByUser(ctx context.Context, userID int, offset, limit int) ([]*models.Article, int, error) {
	articles, total, err := s.articleRepo.ListByUser(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleCategories(ctx, articles); err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleTags(ctx, articles); err != nil {
		return nil, 0, err
	}
	s.fillUpvoteCounts(ctx, articles)
	return articles, total, nil
}

func (s *ArticleService) GetTags(ctx context.Context, articleID int) ([]*models.Tag, error) {
	return s.articleTagRepo.GetByArticle(ctx, articleID)
}

func (s *ArticleService) GetCategories(ctx context.Context, articleID int) ([]*models.Category, error) {
	return s.articleCategoryRepo.GetByArticle(ctx, articleID)
}

// PublicList 公开文章列表（仅已发布文章，支持分类/标签 slug 筛选）
func (s *ArticleService) PublicList(ctx context.Context, offset, limit int, categorySlug, tagSlug string) ([]*models.Article, int, error) {
	articles, total, err := s.articleRepo.PublicList(ctx, offset, limit, categorySlug, tagSlug)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleCategories(ctx, articles); err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleTags(ctx, articles); err != nil {
		return nil, 0, err
	}
	s.fillUpvoteCounts(ctx, articles)
	return articles, total, nil
}

// Search searches published articles by keyword.
func (s *ArticleService) Search(ctx context.Context, keyword string, offset, limit int) ([]*models.Article, int, error) {
	articles, total, err := s.articleRepo.Search(ctx, keyword, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleCategories(ctx, articles); err != nil {
		return nil, 0, err
	}
	if err := s.fillArticleTags(ctx, articles); err != nil {
		return nil, 0, err
	}
	s.fillUpvoteCounts(ctx, articles)
	return articles, total, nil
}

func (s *ArticleService) fillArticleCategories(ctx context.Context, articles []*models.Article) error {
	if len(articles) == 0 {
		return nil
	}
	ids := make([]int, len(articles))
	for i, a := range articles {
		ids[i] = a.ID
	}
	categoriesMap, err := s.articleCategoryRepo.GetByArticles(ctx, ids)
	if err != nil {
		return err
	}
	for _, a := range articles {
		a.Categories = categoriesMap[a.ID]
	}
	return nil
}

func (s *ArticleService) fillArticleTags(ctx context.Context, articles []*models.Article) error {
	if len(articles) == 0 {
		return nil
	}
	ids := make([]int, len(articles))
	for i, a := range articles {
		ids[i] = a.ID
	}
	tagsMap, err := s.articleTagRepo.GetByArticles(ctx, ids)
	if err != nil {
		return err
	}
	for _, a := range articles {
		a.Tags = tagsMap[a.ID]
	}
	return nil
}
