package repository

import (
	"context"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/sqlc/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	*db.Queries
	Pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{Queries: db.New(pool), Pool: pool}
}

type UserRepositoryImpl struct {
	db *DB
}

func NewUserRepository(database *DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: database}
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *models.User) error {
	result, err := r.db.CreateUser(ctx, db.CreateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Nickname:     toNullString(user.Nickname),
		Avatar:       toNullString(user.Avatar),
		Bio:          toNullText(user.Bio),
		Group:        toNullString(user.Group),
		Status:       toNullInt4(user.Status),
	})
	if err != nil {
		return err
	}
	user.ID = int(result.ID)
	return nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id int) (*models.User, error) {
	result, err := r.db.GetUserByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	return toUserModel(result), nil
}

func (r *UserRepositoryImpl) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	result, err := r.db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return toUserModel(result), nil
}

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	result, err := r.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return toUserModel(result), nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *models.User) error {
	return r.db.UpdateUser(ctx, db.UpdateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Nickname:     toNullString(user.Nickname),
		Avatar:       toNullString(user.Avatar),
		Bio:          toNullText(user.Bio),
		Group:        toNullString(user.Group),
		Status:       toNullInt4(user.Status),
		ID:           int32(user.ID),
	})
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, id int) error {
	return r.db.DeleteUser(ctx, int32(id))
}

func (r *UserRepositoryImpl) List(ctx context.Context, offset, limit int) ([]*models.User, int, error) {
	users, err := r.db.ListUsers(ctx, db.ListUsersParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, 0, err
	}
	count, err := r.db.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*models.User, len(users))
	for i, u := range users {
		result[i] = toUserModel(u)
	}
	return result, int(count), nil
}

func (r *UserRepositoryImpl) UpdateStatus(ctx context.Context, id int, status int) error {
	return r.db.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		Status: pgtype.Int4{Int32: int32(status), Valid: true},
		ID:     int32(id),
	})
}

type ArticleRepositoryImpl struct {
	db *DB
}

func NewArticleRepository(database *DB) *ArticleRepositoryImpl {
	return &ArticleRepositoryImpl{db: database}
}

func (r *ArticleRepositoryImpl) Create(ctx context.Context, article *models.Article) error {
	result, err := r.db.CreateArticle(ctx, db.CreateArticleParams{
		Title:          article.Title,
		Slug:           article.Slug,
		Summary:        toNullText(article.Summary),
		Content:        article.Content,
		CoverUrl:       toNullText(article.CoverURL),
		UserID:         int32(article.UserID),
		Status:         toNullString(article.Status),
		CommentEnabled: pgtype.Bool{Bool: article.CommentEnabled, Valid: true},
		ViewCount:      int32(article.ViewCount),
	})
	if err != nil {
		return err
	}
	article.ID = int(result.ID)
	return nil
}

func (r *ArticleRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Article, error) {
	result, err := r.db.GetArticleByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	return toArticleModel(result), nil
}

func (r *ArticleRepositoryImpl) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	result, err := r.db.GetArticleBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toArticleModel(result), nil
}

func (r *ArticleRepositoryImpl) Update(ctx context.Context, article *models.Article) error {
	return r.db.UpdateArticle(ctx, db.UpdateArticleParams{
		Title:          article.Title,
		Slug:           article.Slug,
		Summary:        toNullText(article.Summary),
		Content:        article.Content,
		CoverUrl:       toNullText(article.CoverURL),
		Status:         toNullString(article.Status),
		CommentEnabled: pgtype.Bool{Bool: article.CommentEnabled, Valid: true},
		ID:             int32(article.ID),
	})
}

func (r *ArticleRepositoryImpl) Delete(ctx context.Context, id int) error {
	return r.db.DeleteArticle(ctx, int32(id))
}

func (r *ArticleRepositoryImpl) List(ctx context.Context, offset, limit int) ([]*models.Article, int, error) {
	articles, err := r.db.ListArticles(ctx, db.ListArticlesParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, 0, err
	}
	count, err := r.db.CountArticles(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*models.Article, len(articles))
	for i, a := range articles {
		result[i] = toArticleModel(a)
	}
	return result, int(count), nil
}

func (r *ArticleRepositoryImpl) ListByUser(ctx context.Context, userID int, offset, limit int) ([]*models.Article, int, error) {
	articles, err := r.db.ListArticlesByUser(ctx, db.ListArticlesByUserParams{
		UserID: int32(userID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	count, err := r.db.CountArticles(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*models.Article, len(articles))
	for i, a := range articles {
		result[i] = toArticleModel(a)
	}
	return result, int(count), nil
}

func (r *ArticleRepositoryImpl) IncrementViewCount(ctx context.Context, id int) error {
	return r.db.IncrementViewCount(ctx, int32(id))
}

// PublicList 公开文章列表（仅未删除的文章，支持 category/tag slug 筛选）
func (r *ArticleRepositoryImpl) PublicList(ctx context.Context, offset, limit int, categorySlug, tagSlug string) ([]*models.Article, int, error) {
	var articles []db.Article
	var count int64

	if tagSlug != "" {
		rows, err := r.db.Pool.Query(ctx,
			`SELECT a.id, a.title, a.slug, a.summary, a.content, a.cover_url, a.user_id, a.status, a.comment_enabled, a.view_count, a.created_at, a.updated_at FROM articles a
			JOIN article_tags at ON a.id = at.article_id
			JOIN tags t ON at.tag_id = t.id
			WHERE a.deleted_at IS NULL AND t.slug = $1
			ORDER BY a.created_at DESC
			LIMIT $2 OFFSET $3`,
			tagSlug, int32(limit), int32(offset))
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		for rows.Next() {
			var a db.Article
			if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Summary, &a.Content, &a.CoverUrl, &a.UserID, &a.Status, &a.CommentEnabled, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
				return nil, 0, err
			}
			articles = append(articles, a)
		}
		r.db.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM articles a
			JOIN article_tags at ON a.id = at.article_id
			JOIN tags t ON at.tag_id = t.id
			WHERE a.deleted_at IS NULL AND t.slug = $1`,
			tagSlug).Scan(&count)
	} else if categorySlug != "" {
		rows, err := r.db.Pool.Query(ctx,
			`SELECT a.id, a.title, a.slug, a.summary, a.content, a.cover_url, a.user_id, a.status, a.comment_enabled, a.view_count, a.created_at, a.updated_at FROM articles a
			JOIN article_categories ac ON a.id = ac.article_id
			JOIN categories c ON ac.category_id = c.id
			WHERE a.deleted_at IS NULL AND c.slug = $1
			ORDER BY a.created_at DESC
			LIMIT $2 OFFSET $3`,
			categorySlug, int32(limit), int32(offset))
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		for rows.Next() {
			var a db.Article
			if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Summary, &a.Content, &a.CoverUrl, &a.UserID, &a.Status, &a.CommentEnabled, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
				return nil, 0, err
			}
			articles = append(articles, a)
		}
		r.db.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM articles a
			JOIN article_categories ac ON a.id = ac.article_id
			JOIN categories c ON ac.category_id = c.id
			WHERE a.deleted_at IS NULL AND c.slug = $1`,
			categorySlug).Scan(&count)
	} else {
		var err error
		articles, err = r.db.ListArticles(ctx, db.ListArticlesParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM articles WHERE deleted_at IS NULL").Scan(&count)
		result := make([]*models.Article, len(articles))
		for i, a := range articles {
			result[i] = toArticleModel(a)
		}
		return result, int(count), nil
	}

	result := make([]*models.Article, len(articles))
	for i, a := range articles {
		result[i] = toArticleModel(a)
	}
	return result, int(count), nil
}

// Search performs a full-text-like search on published articles (title, summary, content).
func (r *ArticleRepositoryImpl) Search(ctx context.Context, keyword string, offset, limit int) ([]*models.Article, int, error) {
	pattern := "%" + keyword + "%"
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at FROM articles
		 WHERE deleted_at IS NULL AND status = 'published'
		 AND (title ILIKE $1 OR summary ILIKE $1 OR content ILIKE $1)
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		pattern, int32(limit), int32(offset))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []db.Article
	for rows.Next() {
		var a db.Article
		if err := rows.Scan(&a.ID, &a.Title, &a.Slug, &a.Summary, &a.Content, &a.CoverUrl, &a.UserID, &a.Status, &a.CommentEnabled, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		articles = append(articles, a)
	}

	var count int64
	err = r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM articles
		 WHERE deleted_at IS NULL AND status = 'published'
		 AND (title ILIKE $1 OR summary ILIKE $1 OR content ILIKE $1)`,
		pattern).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*models.Article, len(articles))
	for i, a := range articles {
		result[i] = toArticleModel(a)
	}
	return result, int(count), nil
}

type MenuRepositoryImpl struct {
	db *DB
}

func NewMenuRepository(database *DB) *MenuRepositoryImpl {
	return &MenuRepositoryImpl{db: database}
}

func (r *MenuRepositoryImpl) GetByName(ctx context.Context, name string) (*models.Menu, error) {
	var id int32
	var menuName string
	err := r.db.Pool.QueryRow(ctx,
		"SELECT id, name FROM menus WHERE name = $1 LIMIT 1", name).Scan(&id, &menuName)
	if err != nil {
		return nil, err
	}
	return &models.Menu{ID: int(id), Name: menuName}, nil
}

func (r *MenuRepositoryImpl) GetItems(ctx context.Context, menuID int) ([]*models.MenuItem, error) {
	rows, err := r.db.Pool.Query(ctx,
		"SELECT id, menu_id, name, url, parent_id, priority FROM menu_items WHERE menu_id = $1 ORDER BY priority ASC, id ASC",
		int32(menuID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.MenuItem
	for rows.Next() {
		var it models.MenuItem
		var parentID pgtype.Int4
		if err := rows.Scan(&it.ID, &it.MenuID, &it.Name, &it.URL, &parentID, &it.Priority); err != nil {
			continue
		}
		if parentID.Valid {
			pid := int(parentID.Int32)
			it.ParentID = &pid
		}
		items = append(items, &it)
	}
	return items, nil
}
