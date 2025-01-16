-- name: StoreRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    NULL
)
RETURNING *;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens 
SET revoked_at = $2, updated_at = $2 WHERE token = $1
RETURNING *; 

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;

