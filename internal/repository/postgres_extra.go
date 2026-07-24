package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/sqlc/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func toNullString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toNullText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toNullInt4(i int) pgtype.Int4 {
	if i == 0 {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

type ArticleUpvoteRepositoryImpl struct {
	db *DB
}

func NewArticleUpvoteRepository(database *DB) *ArticleUpvoteRepositoryImpl {
	return &ArticleUpvoteRepositoryImpl{db: database}
}

func (r *ArticleUpvoteRepositoryImpl) Toggle(ctx context.Context, articleID int, userID int, ip string) (int, error) {
	var exists bool
	if userID > 0 {
		err := r.db.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM article_upvotes WHERE article_id = $1 AND user_id = $2)",
			int32(articleID), int32(userID)).Scan(&exists)
		if err != nil {
			return 0, err
		}
		if exists {
			_, err = r.db.Pool.Exec(ctx,
				"DELETE FROM article_upvotes WHERE article_id = $1 AND user_id = $2",
				int32(articleID), int32(userID))
		} else {
			_, err = r.db.Pool.Exec(ctx,
				"INSERT INTO article_upvotes (article_id, user_id, created_at) VALUES ($1, $2, NOW())",
				int32(articleID), int32(userID))
		}
		if err != nil {
			return 0, err
		}
	} else if ip != "" {
		err := r.db.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM article_upvotes WHERE article_id = $1 AND ip_address = $2)",
			int32(articleID), ip).Scan(&exists)
		if err != nil {
			return 0, err
		}
		if exists {
			_, err = r.db.Pool.Exec(ctx,
				"DELETE FROM article_upvotes WHERE article_id = $1 AND ip_address = $2",
				int32(articleID), ip)
		} else {
			_, err = r.db.Pool.Exec(ctx,
				"INSERT INTO article_upvotes (article_id, ip_address, created_at) VALUES ($1, $2, NOW())",
				int32(articleID), ip)
		}
		if err != nil {
			return 0, err
		}
	}
	return r.Count(ctx, articleID)
}

func (r *ArticleUpvoteRepositoryImpl) Count(ctx context.Context, articleID int) (int, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM article_upvotes WHERE article_id = $1", int32(articleID)).Scan(&count)
	return int(count), err
}

func (r *ArticleUpvoteRepositoryImpl) CountByArticles(ctx context.Context, articleIDs []int) (map[int]int, error) {
	result := make(map[int]int)
	if len(articleIDs) == 0 {
		return result, nil
	}
	ids := make([]int32, len(articleIDs))
	for i, id := range articleIDs {
		ids[i] = int32(id)
	}
	rows, err := r.db.Pool.Query(ctx,
		"SELECT article_id, COUNT(*) FROM article_upvotes WHERE article_id = ANY($1) GROUP BY article_id", ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var aid int64
		var cnt int64
		if err := rows.Scan(&aid, &cnt); err != nil {
			return nil, err
		}
		result[int(aid)] = int(cnt)
	}
	return result, nil
}

func toUserModel(u db.User) *models.User {
	user := &models.User{
		ID:           int(u.ID),
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Nickname:     u.Nickname.String,
		Avatar:       u.Avatar.String,
		Bio:          u.Bio.String,
		Group:        u.Group.String,
		Status:       int(u.Status.Int32),
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
	return user
}

func toArticleModel(a db.Article) *models.Article {
	article := &models.Article{
		ID:             int(a.ID),
		Title:          a.Title,
		Slug:           a.Slug,
		Content:        a.Content,
		Status:         a.Status.String,
		CommentEnabled: a.CommentEnabled.Bool,
		UserID:         int(a.UserID),
		ViewCount:      int(a.ViewCount),
		CreatedAt:      a.CreatedAt.Time,
		UpdatedAt:      a.UpdatedAt.Time,
	}
	if a.Summary.Valid {
		article.Summary = a.Summary.String
	}
	if a.CoverUrl.Valid {
		article.CoverURL = a.CoverUrl.String
	}
	if a.DeletedAt.Valid {
		article.DeletedAt = &a.DeletedAt.Time
	}
	return article
}

type CategoryRepositoryImpl struct {
	db *DB
}

func NewCategoryRepository(database *DB) *CategoryRepositoryImpl {
	return &CategoryRepositoryImpl{db: database}
}

func (r *CategoryRepositoryImpl) Create(ctx context.Context, category *models.Category) error {
	result, err := r.db.CreateCategory(ctx, db.CreateCategoryParams{
		Name:        category.Name,
		Slug:        category.Slug,
		Description: toNullText(category.Description),
		Icon:        toNullText(category.Icon),
	})
	if err != nil {
		return err
	}
	category.ID = int(result.ID)
	category.Description = result.Description.String
	category.Icon = result.Icon.String
	return nil
}

func (r *CategoryRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Category, error) {
	result, err := r.db.GetCategoryByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	return toCategoryModel(db.Category{
		ID:          result.ID,
		Name:        result.Name,
		Slug:        result.Slug,
		Description: result.Description,
		Icon:        result.Icon,
		CreatedAt:   result.CreatedAt,
	}), nil
}

func (r *CategoryRepositoryImpl) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	result, err := r.db.GetCategoryBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toCategoryModel(db.Category{
		ID:          result.ID,
		Name:        result.Name,
		Slug:        result.Slug,
		Description: result.Description,
		Icon:        result.Icon,
		CreatedAt:   result.CreatedAt,
	}), nil
}

func (r *CategoryRepositoryImpl) Update(ctx context.Context, category *models.Category) error {
	err := r.db.UpdateCategory(ctx, db.UpdateCategoryParams{
		Name:        category.Name,
		Slug:        category.Slug,
		Description: toNullText(category.Description),
		Icon:        toNullText(category.Icon),
		ID:          int32(category.ID),
	})
	return err
}

func (r *CategoryRepositoryImpl) Delete(ctx context.Context, id int) error {
	return r.db.DeleteCategory(ctx, int32(id))
}

func (r *CategoryRepositoryImpl) List(ctx context.Context) ([]*models.Category, error) {
	categories, err := r.db.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.Category, len(categories))
	for i, c := range categories {
		result[i] = toCategoryModel(db.Category{
			ID:          c.ID,
			Name:        c.Name,
			Slug:        c.Slug,
			Description: c.Description,
			Icon:        c.Icon,
			CreatedAt:   c.CreatedAt,
		})
	}
	return result, nil
}

func (r *CategoryRepositoryImpl) GetArticleCount(ctx context.Context, categoryID int) (int, error) {
	count, err := r.db.CountArticlesByCategory(ctx, int32(categoryID))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func toCategoryModel(c db.Category) *models.Category {
	return &models.Category{
		ID:          int(c.ID),
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description.String,
		Icon:        c.Icon.String,
		CreatedAt:   c.CreatedAt.Time,
	}
}

type TagRepositoryImpl struct {
	db *DB
}

func NewTagRepository(database *DB) *TagRepositoryImpl {
	return &TagRepositoryImpl{db: database}
}

func (r *TagRepositoryImpl) Create(ctx context.Context, tag *models.Tag) error {
	result, err := r.db.CreateTag(ctx, db.CreateTagParams{
		Name: tag.Name,
		Slug: tag.Slug,
		Icon: toNullText(tag.Icon),
	})
	if err != nil {
		return err
	}
	tag.ID = int(result.ID)
	tag.Icon = result.Icon.String
	return nil
}

func (r *TagRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Tag, error) {
	result, err := r.db.GetTagByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	return toTagModel(result), nil
}

func (r *TagRepositoryImpl) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	result, err := r.db.GetTagBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toTagModel(result), nil
}

func (r *TagRepositoryImpl) Update(ctx context.Context, tag *models.Tag) error {
	err := r.db.UpdateTag(ctx, db.UpdateTagParams{
		Name: tag.Name,
		Slug: tag.Slug,
		Icon: toNullText(tag.Icon),
		ID:   int32(tag.ID),
	})
	return err
}

func (r *TagRepositoryImpl) Delete(ctx context.Context, id int) error {
	return r.db.DeleteTag(ctx, int32(id))
}

func (r *TagRepositoryImpl) List(ctx context.Context) ([]*models.Tag, error) {
	tags, err := r.db.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.Tag, len(tags))
	for i, t := range tags {
		result[i] = toTagModel(t)
	}
	return result, nil
}

func toTagModel(t db.Tag) *models.Tag {
	return &models.Tag{
		ID:   int(t.ID),
		Name: t.Name,
		Slug: t.Slug,
		Icon: t.Icon.String,
	}
}

type ArticleTagRepositoryImpl struct {
	db *DB
}

func NewArticleTagRepository(database *DB) *ArticleTagRepositoryImpl {
	return &ArticleTagRepositoryImpl{db: database}
}

func (r *ArticleTagRepositoryImpl) Create(ctx context.Context, articleID, tagID int) error {
	return r.db.CreateArticleTag(ctx, db.CreateArticleTagParams{
		ArticleID: int32(articleID),
		TagID:     int32(tagID),
	})
}

func (r *ArticleTagRepositoryImpl) DeleteByArticle(ctx context.Context, articleID int) error {
	return r.db.DeleteArticleTags(ctx, int32(articleID))
}

func (r *ArticleTagRepositoryImpl) GetByArticle(ctx context.Context, articleID int) ([]*models.Tag, error) {
	tags, err := r.db.GetTagsByArticle(ctx, int32(articleID))
	if err != nil {
		return nil, err
	}
	result := make([]*models.Tag, len(tags))
	for i, t := range tags {
		result[i] = toTagModel(t)
	}
	return result, nil
}

func (r *ArticleTagRepositoryImpl) GetByArticles(ctx context.Context, articleIDs []int) (map[int][]*models.Tag, error) {
	if len(articleIDs) == 0 {
		return map[int][]*models.Tag{}, nil
	}
	ids := make([]int32, len(articleIDs))
	for i, id := range articleIDs {
		ids[i] = int32(id)
	}
	rows, err := r.db.Pool.Query(ctx,
		`SELECT at.article_id, t.id, t.name, t.slug FROM tags t
		 INNER JOIN article_tags at ON t.id = at.tag_id
		 WHERE at.article_id = ANY($1)
		 ORDER BY t.name`,
		ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]*models.Tag)
	for rows.Next() {
		var articleID int32
		var t models.Tag
		if err := rows.Scan(&articleID, &t.ID, &t.Name, &t.Slug); err != nil {
			return nil, err
		}
		aid := int(articleID)
		result[aid] = append(result[aid], &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// CountAll returns the number of non-deleted articles for each tag.
func (r *ArticleTagRepositoryImpl) CountAll(ctx context.Context) (map[int]int, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT at.tag_id, COUNT(*) FROM article_tags at
		 INNER JOIN articles a ON a.id = at.article_id
		 WHERE a.deleted_at IS NULL
		 GROUP BY at.tag_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]int)
	for rows.Next() {
		var tagID int64
		var count int64
		if err := rows.Scan(&tagID, &count); err != nil {
			return nil, err
		}
		result[int(tagID)] = int(count)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type ArticleCategoryRepositoryImpl struct {
	db *DB
}

func NewArticleCategoryRepository(database *DB) *ArticleCategoryRepositoryImpl {
	return &ArticleCategoryRepositoryImpl{db: database}
}

func (r *ArticleCategoryRepositoryImpl) Create(ctx context.Context, articleID, categoryID int) error {
	return r.db.CreateArticleCategory(ctx, db.CreateArticleCategoryParams{
		ArticleID:  int32(articleID),
		CategoryID: int32(categoryID),
	})
}

func (r *ArticleCategoryRepositoryImpl) DeleteByArticle(ctx context.Context, articleID int) error {
	return r.db.DeleteArticleCategories(ctx, int32(articleID))
}

func (r *ArticleCategoryRepositoryImpl) GetByArticle(ctx context.Context, articleID int) ([]*models.Category, error) {
	categories, err := r.db.GetCategoriesByArticle(ctx, int32(articleID))
	if err != nil {
		return nil, err
	}
	result := make([]*models.Category, len(categories))
	for i, c := range categories {
		result[i] = toCategoryModel(c)
	}
	return result, nil
}

func (r *ArticleCategoryRepositoryImpl) GetByArticles(ctx context.Context, articleIDs []int) (map[int][]*models.Category, error) {
	if len(articleIDs) == 0 {
		return map[int][]*models.Category{}, nil
	}
	ids := make([]int32, len(articleIDs))
	for i, id := range articleIDs {
		ids[i] = int32(id)
	}
	rows, err := r.db.Pool.Query(ctx,
		`SELECT ac.article_id, c.id, c.name, c.slug, c.description, c.icon FROM categories c
		 INNER JOIN article_categories ac ON c.id = ac.category_id
		 WHERE ac.article_id = ANY($1)
		 ORDER BY c.name`,
		ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]*models.Category)
	for rows.Next() {
		var articleID int32
		var c models.Category
		if err := rows.Scan(&articleID, &c.ID, &c.Name, &c.Slug, &c.Description, &c.Icon); err != nil {
			return nil, err
		}
		aid := int(articleID)
		result[aid] = append(result[aid], &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type CommentRepositoryImpl struct {
	db *DB
}

func NewCommentRepository(database *DB) *CommentRepositoryImpl {
	return &CommentRepositoryImpl{db: database}
}

func (r *CommentRepositoryImpl) Create(ctx context.Context, comment *models.Comment) error {
	var parentID pgtype.Int4
	if comment.ParentID != nil {
		parentID = pgtype.Int4{Int32: int32(*comment.ParentID), Valid: true}
	}
	var userID pgtype.Int4
	if comment.UserID != nil {
		userID = pgtype.Int4{Int32: int32(*comment.UserID), Valid: true}
	}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO comments (article_id, parent_id, user_id, author_name, author_email, author_url, author_avatar, content, status, ip_address, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		 RETURNING id, created_at`,
		int32(comment.ArticleID), parentID, userID,
		toNullString(comment.AuthorName), toNullString(comment.AuthorEmail), toNullString(comment.AuthorURL), toNullString(comment.AuthorAvatar),
		comment.Content, comment.Status, toNullString(comment.IPAddress),
	).Scan(&comment.ID, &comment.CreatedAt)
	return err
}

func (r *CommentRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Comment, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, article_id, parent_id, user_id, author_name, author_email, author_url, author_avatar, content, status, ip_address, created_at, deleted_at
		 FROM comments WHERE id = $1`, int32(id))
	return scanCommentRow(row)
}

func (r *CommentRepositoryImpl) Delete(ctx context.Context, id int) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM comments WHERE id = $1", int32(id))
	return err
}

func (r *CommentRepositoryImpl) ListByArticle(ctx context.Context, articleID int, offset, limit int) ([]*models.Comment, int, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, article_id, parent_id, user_id, author_name, author_email, author_url, author_avatar, content, status, ip_address, created_at, deleted_at
		 FROM comments
		 WHERE article_id = $1 AND deleted_at IS NULL AND status = 'approved'
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		int32(articleID), int32(limit), int32(offset))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		c, err := scanCommentRows(rows)
		if err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}

	var count int64
	err = r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM comments WHERE article_id = $1 AND deleted_at IS NULL AND status = 'approved'",
		int32(articleID)).Scan(&count)
	if err != nil {
		return nil, 0, err
	}
	return comments, int(count), nil
}

func (r *CommentRepositoryImpl) CountByArticles(ctx context.Context, articleIDs []int) (map[int]int, error) {
	result := make(map[int]int)
	if len(articleIDs) == 0 {
		return result, nil
	}
	ids := make([]int32, len(articleIDs))
	for i, id := range articleIDs {
		ids[i] = int32(id)
	}
	rows, err := r.db.Pool.Query(ctx,
		`SELECT article_id, COUNT(*) FROM comments
		 WHERE article_id = ANY($1) AND deleted_at IS NULL AND status = 'approved'
		 GROUP BY article_id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var aid int64
		var cnt int64
		if err := rows.Scan(&aid, &cnt); err != nil {
			return nil, err
		}
		result[int(aid)] = int(cnt)
	}
	return result, nil
}

func (r *CommentRepositoryImpl) List(ctx context.Context, offset, limit int, status string) ([]*models.Comment, int, error) {
	var filterArgs []any
	where := "deleted_at IS NULL"
	if status != "" && status != "all" {
		where += " AND status = $1"
		filterArgs = append(filterArgs, status)
	}

	// Build query with sequential parameter numbers.
	var query string
	var queryArgs []any
	if len(filterArgs) > 0 {
		query = "SELECT id, article_id, parent_id, user_id, author_name, author_email, author_url, author_avatar, content, status, ip_address, created_at, deleted_at FROM comments WHERE " + where + " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		queryArgs = append(queryArgs, filterArgs...)
	} else {
		query = "SELECT id, article_id, parent_id, user_id, author_name, author_email, author_url, author_avatar, content, status, ip_address, created_at, deleted_at FROM comments WHERE " + where + " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
	}
	queryArgs = append(queryArgs, int32(limit), int32(offset))

	rows, err := r.db.Pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		c, err := scanCommentRows(rows)
		if err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}

	countQuery := "SELECT COUNT(*) FROM comments WHERE " + where
	var count int64
	err = r.db.Pool.QueryRow(ctx, countQuery, filterArgs...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}
	return comments, int(count), nil
}

func (r *CommentRepositoryImpl) UpdateStatus(ctx context.Context, id int, status string) error {
	_, err := r.db.Pool.Exec(ctx, "UPDATE comments SET status = $1 WHERE id = $2", status, int32(id))
	return err
}

type commentScanner interface {
	Scan(dest ...any) error
}

func scanCommentRow(sc commentScanner) (*models.Comment, error) {
	var c models.Comment
	var parentID, userID pgtype.Int4
	var authorName, authorEmail, authorURL, authorAvatar, status, ipAddress pgtype.Text
	var deletedAt pgtype.Timestamp
	err := sc.Scan(
		&c.ID, &c.ArticleID, &parentID, &userID,
		&authorName, &authorEmail, &authorURL, &authorAvatar,
		&c.Content, &status, &ipAddress,
		&c.CreatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		pid := int(parentID.Int32)
		c.ParentID = &pid
	}
	if userID.Valid {
		uid := int(userID.Int32)
		c.UserID = &uid
	}
	c.AuthorName = authorName.String
	c.AuthorEmail = authorEmail.String
	c.AuthorURL = authorURL.String
	c.AuthorAvatar = authorAvatar.String
	c.Status = status.String
	if c.Status == "" {
		c.Status = "approved"
	}
	c.IPAddress = ipAddress.String
	if deletedAt.Valid {
		t := deletedAt.Time
		c.DeletedAt = &t
	}
	return &c, nil
}

func scanCommentRows(rows interface {
	Scan(dest ...any) error
	Err() error
}) (*models.Comment, error) {
	return scanCommentRow(rows)
}

type SettingRepositoryImpl struct {
	db *DB
}

func NewSettingRepository(database *DB) *SettingRepositoryImpl {
	return &SettingRepositoryImpl{db: database}
}

func (r *SettingRepositoryImpl) Get(ctx context.Context, key string) (*models.Setting, error) {
	result, err := r.db.GetSetting(ctx, key)
	if err != nil {
		return nil, err
	}
	return toSettingModel(result), nil
}

func (r *SettingRepositoryImpl) Set(ctx context.Context, key, value, description string) error {
	jsonValue, _ := json.Marshal(value)
	return r.db.SetSetting(ctx, db.SetSettingParams{
		Key:         key,
		Value:       jsonValue,
		Description: toNullString(description),
	})
}

func (r *SettingRepositoryImpl) List(ctx context.Context) ([]*models.Setting, error) {
	settings, err := r.db.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*models.Setting, len(settings))
	for i, s := range settings {
		result[i] = toSettingModel(s)
	}
	return result, nil
}

func toSettingModel(s db.Setting) *models.Setting {
	setting := &models.Setting{
		Key: s.Key,
	}
	if s.Value != nil {
		json.Unmarshal(s.Value, &setting.Value)
	}
	if s.Description.Valid {
		setting.Description = s.Description.String
	}
	return setting
}

var _ = time.Time{}
