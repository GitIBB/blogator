-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, url, name, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING * 
)
SELECT
    inserted_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN feeds ON inserted_feed_follow.feed_id = feeds.id
INNER JOIN users ON inserted_feed_follow.user_id = users.id
;

-- name: GetFeedByURL :one
SELECT *
FROM feeds
WHERE url = $1
LIMIT 1;

-- name: GetFeedFollowsForUser :many
SELECT feed_follows.id,
       feed_follows.user_id,
       feed_follows.feed_id,
       feed_follows.created_at,
       feed_follows.updated_at,
       feeds.name AS feed_name,
       users.name AS user_name
FROM feed_follows
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
INNER JOIN users ON feed_follows.user_id = users.id
WHERE feed_follows.user_id = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
USING feeds
WHERE feed_follows.user_id = $1 
    AND feed_follows.feed_id = feeds.id
    AND feeds.url = $2;


-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = NOW(), updated_at = NOW()
WHERE feeds.id = $1;


-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;

-- name: GetFeedCreator :many
SELECT feeds.*, users.name as user_name
FROM feeds
JOIN users ON feeds.user_id = users.id;