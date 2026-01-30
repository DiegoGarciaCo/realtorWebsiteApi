-- name: CheckSessionByID :one
SELECT
    "userId",
    "expiresAt"
FROM
    SESSION
WHERE
    token = $1;
