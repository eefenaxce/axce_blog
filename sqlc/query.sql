-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, nickname, avatar, bio, "group", status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
RETURNING id, username, email, password_hash, nickname, avatar, bio, "group", status, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, nickname, avatar, bio, "group", status, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, email, password_hash, nickname, avatar, bio, "group", status, created_at, updated_at
FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, nickname, avatar, bio, "group", status, created_at, updated_at
FROM users WHERE email = $1;

-- name: UpdateUser :exec
UPDATE users SET username=$1, email=$2, password_hash=$3, nickname=$4, avatar=$5, bio=$6, "group"=$7, status=$8, updated_at=NOW()
WHERE id=$9;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, password_hash, nickname, avatar, bio, "group", status, created_at, updated_at
FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: UpdateUserStatus :exec
UPDATE users SET status=$1, updated_at=NOW() WHERE id=$2;

-- name: CreateArticle :one
INSERT INTO articles (title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING id, title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at, deleted_at;

-- name: GetArticleByID :one
SELECT id, title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at, deleted_at
FROM articles WHERE id = $1 AND deleted_at IS NULL;

-- name: GetArticleBySlug :one
SELECT id, title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at, deleted_at
FROM articles WHERE slug = $1 AND deleted_at IS NULL;

-- name: UpdateArticle :exec
UPDATE articles SET title=$1, slug=$2, summary=$3, content=$4, cover_url=$5, status=$6, comment_enabled=$7, updated_at=NOW()
WHERE id=$8 AND deleted_at IS NULL;

-- name: DeleteArticle :exec
DELETE FROM articles WHERE id = $1;

-- name: ListArticles :many
SELECT id, title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at, deleted_at
FROM articles WHERE deleted_at IS NULL
ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListArticlesByCategory :many
SELECT a.id, a.title, a.slug, a.summary, a.content, a.cover_url, a.user_id, a.status, a.comment_enabled, a.view_count, a.created_at, a.updated_at, a.deleted_at
FROM articles a
INNER JOIN article_categories ac ON a.id = ac.article_id
WHERE ac.category_id = $1 AND a.deleted_at IS NULL
ORDER BY a.created_at DESC LIMIT $2 OFFSET $3;

-- name: ListArticlesByUser :many
SELECT id, title, slug, summary, content, cover_url, user_id, status, comment_enabled, view_count, created_at, updated_at, deleted_at
FROM articles WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountArticles :one
SELECT COUNT(*) FROM articles WHERE deleted_at IS NULL;

-- name: CountArticlesByCategory :one
SELECT COUNT(*) FROM article_categories ac
INNER JOIN articles a ON ac.article_id = a.id
WHERE ac.category_id = $1 AND a.deleted_at IS NULL;

-- name: IncrementViewCount :exec
UPDATE articles SET view_count = view_count + 1 WHERE id = $1;

-- name: CreateCategory :one
INSERT INTO categories (name, slug, description, icon, created_at) VALUES ($1, $2, $3, $4, NOW()) RETURNING id, name, slug, description, icon, created_at;

-- name: GetCategoryByID :one
SELECT id, name, slug, description, icon, created_at FROM categories WHERE id = $1;

-- name: GetCategoryBySlug :one
SELECT id, name, slug, description, icon, created_at FROM categories WHERE slug = $1;

-- name: UpdateCategory :exec
UPDATE categories SET name=$1, slug=$2, description=$3, icon=$4 WHERE id=$5;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-- name: ListCategories :many
SELECT id, name, slug, description, icon, created_at FROM categories ORDER BY name;

-- name: CreateTag :one
INSERT INTO tags (name, slug, icon) VALUES ($1, $2, $3) RETURNING id, name, slug, icon;

-- name: GetTagByID :one
SELECT id, name, slug, icon FROM tags WHERE id = $1;

-- name: GetTagBySlug :one
SELECT id, name, slug, icon FROM tags WHERE slug = $1;

-- name: UpdateTag :exec
UPDATE tags SET name=$1, slug=$2, icon=$3 WHERE id=$4;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- name: ListTags :many
SELECT id, name, slug, icon FROM tags ORDER BY name;

-- name: CreateArticleTag :exec
INSERT INTO article_tags (article_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteArticleTags :exec
DELETE FROM article_tags WHERE article_id = $1;

-- name: GetTagsByArticle :many
SELECT t.id, t.name, t.slug, t.icon FROM tags t
INNER JOIN article_tags at ON t.id = at.tag_id
WHERE at.article_id = $1;

-- name: CreateArticleCategory :exec
INSERT INTO article_categories (article_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteArticleCategories :exec
DELETE FROM article_categories WHERE article_id = $1;

-- name: GetCategoriesByArticle :many
SELECT c.id, c.name, c.slug, c.description, c.icon, c.created_at FROM categories c
INNER JOIN article_categories ac ON c.id = ac.category_id
WHERE ac.article_id = $1;

-- name: CreateComment :one
INSERT INTO comments (article_id, user_id, content, created_at) VALUES ($1, $2, $3, NOW()) RETURNING id, article_id, user_id, content, created_at, deleted_at;

-- name: GetCommentByID :one
SELECT id, article_id, user_id, content, created_at, deleted_at FROM comments WHERE id = $1;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: ListCommentsByArticle :many
SELECT id, article_id, user_id, content, created_at, deleted_at FROM comments
WHERE article_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountCommentsByArticle :one
SELECT COUNT(*) FROM comments WHERE article_id = $1 AND deleted_at IS NULL;

-- name: GetSetting :one
SELECT key, value, description FROM settings WHERE key = $1;

-- name: SetSetting :exec
INSERT INTO settings (key, value, description) VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET value = $2, description = $3;

-- name: ListSettings :many
SELECT key, value, description FROM settings ORDER BY key;

