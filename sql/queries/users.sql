-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;


-- name: GetUser :one
SELECT * from users WHERE name = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;


-- name: GetUsers :many
SELECT name from users;