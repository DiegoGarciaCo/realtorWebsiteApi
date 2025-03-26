-- name: StoreRefreshToken :exec
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at) VALUES ($1, now(), now(), $2, $3);

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE token = $1;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: GetCsfToken :one
SELECT * FROM csft WHERE token = $1;

-- name: StorecsfToken :exec
INSERT INTO csft (token, created_at, user_id) VALUES ($1, now(), $2);

-- name: DeletecsfToken :exec
DELETE FROM csft WHERE token = $1;