-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (username, first_name, last_name, email, password_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;