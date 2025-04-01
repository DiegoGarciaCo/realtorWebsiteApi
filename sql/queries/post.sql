-- name: ListPublishedPosts :many
SELECT id, title, slug, excerpt, content, created_at, tags, thumbnail, published_at, author
FROM posts
WHERE status = 'published'
ORDER BY created_at DESC;

-- name: GetPostBySlug :one
SELECT id, title, slug, content, excerpt, status, author, published_at, thumbnail, created_at, updated_at, tags
FROM posts
WHERE slug = $1;

-- name: GetPostThumbnailById :one
SELECT id, thumbnail, created_at, updated_at
FROM posts
WHERE id = $1;

-- name: UpdatePostThumbnail :exec
UPDATE posts
SET thumbnail = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreatePost :one
INSERT INTO posts (title, slug, content, excerpt, author, published_at, status, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, title, slug, content, excerpt, status, created_at, updated_at, tags;

-- name: UpdatePost :one
UPDATE posts
SET title = $2, slug = $3, content = $4, excerpt = $5, author = $6, status = $7, tags = $8, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, title, slug, content, excerpt, status, created_at, updated_at, tags;

-- name: SaveAndPublishPost :one
UPDATE posts
SET title = $2, slug = $3, content = $4, excerpt = $5, author = $6, status = $7, tags = $8, updated_at = CURRENT_TIMESTAMP, published_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, title, slug, content, excerpt, status, created_at, updated_at, tags;

-- name: DeletePost :one
DELETE FROM posts
WHERE id = $1 RETURNING id, thumbnail;

-- name: ListAllPosts :many
SELECT id, title, slug, excerpt, content, author, published_at, thumbnail, status, created_at, tags
FROM posts
ORDER BY created_at DESC;

-- name: PublishPost :exec
UPDATE posts
SET status = $2, updated_at = CURRENT_TIMESTAMP, published_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UnpublishPost :exec
UPDATE posts
SET status = "draft", updated_at = CURRENT_TIMESTAMP, published_at = NULL
WHERE id = $1;

