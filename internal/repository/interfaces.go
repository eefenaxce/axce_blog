package repository

import (
	"context"

	"github.com/eefenaxce/axce_blog/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, offset, limit int) ([]*models.User, int, error)
	UpdateStatus(ctx context.Context, id int, status int) error
}

type ArticleRepository interface {
	Create(ctx context.Context, article *models.Article) error
	GetByID(ctx context.Context, id int) (*models.Article, error)
	GetBySlug(ctx context.Context, slug string) (*models.Article, error)
	Update(ctx context.Context, article *models.Article) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, offset, limit int) ([]*models.Article, int, error)
	ListByUser(ctx context.Context, userID int, offset, limit int) ([]*models.Article, int, error)
	IncrementViewCount(ctx context.Context, id int) error
	PublicList(ctx context.Context, offset, limit int, categorySlug, tagSlug string) ([]*models.Article, int, error)
	Search(ctx context.Context, keyword string, offset, limit int) ([]*models.Article, int, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id int) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context) ([]*models.Category, error)
	GetArticleCount(ctx context.Context, categoryID int) (int, error)
}

type TagRepository interface {
	Create(ctx context.Context, tag *models.Tag) error
	GetByID(ctx context.Context, id int) (*models.Tag, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tag, error)
	Update(ctx context.Context, tag *models.Tag) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context) ([]*models.Tag, error)
}

type ArticleTagRepository interface {
	Create(ctx context.Context, articleID, tagID int) error
	DeleteByArticle(ctx context.Context, articleID int) error
	GetByArticle(ctx context.Context, articleID int) ([]*models.Tag, error)
	GetByArticles(ctx context.Context, articleIDs []int) (map[int][]*models.Tag, error)
	CountAll(ctx context.Context) (map[int]int, error)
}

type ArticleCategoryRepository interface {
	Create(ctx context.Context, articleID, categoryID int) error
	DeleteByArticle(ctx context.Context, articleID int) error
	GetByArticle(ctx context.Context, articleID int) ([]*models.Category, error)
	GetByArticles(ctx context.Context, articleIDs []int) (map[int][]*models.Category, error)
}

type CommentRepository interface {
	Create(ctx context.Context, comment *models.Comment) error
	GetByID(ctx context.Context, id int) (*models.Comment, error)
	Delete(ctx context.Context, id int) error
	ListByArticle(ctx context.Context, articleID int, offset, limit int) ([]*models.Comment, int, error)
	CountByArticles(ctx context.Context, articleIDs []int) (map[int]int, error)
	List(ctx context.Context, offset, limit int, status string) ([]*models.Comment, int, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type ArticleUpvoteRepository interface {
	Toggle(ctx context.Context, articleID int, userID int, ip string) (int, error)
	Count(ctx context.Context, articleID int) (int, error)
	CountByArticles(ctx context.Context, articleIDs []int) (map[int]int, error)
}

type SettingRepository interface {
	Get(ctx context.Context, key string) (*models.Setting, error)
	Set(ctx context.Context, key, value, description string) error
	List(ctx context.Context) ([]*models.Setting, error)
}

type MenuRepository interface {
	GetByName(ctx context.Context, name string) (*models.Menu, error)
	GetItems(ctx context.Context, menuID int) ([]*models.MenuItem, error)
}
