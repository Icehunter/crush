-- name: CreateSlice :one
INSERT INTO slices (
    id,
    milestone_id,
    title,
    status,
    phase,
    sort_order,
    depends_on,
    created_at,
    updated_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING *;

-- name: GetSlice :one
SELECT * FROM slices
WHERE id = ? LIMIT 1;

-- name: ListSlicesByMilestone :many
SELECT * FROM slices
WHERE milestone_id = ?
ORDER BY sort_order;

-- name: UpdateSliceStatus :one
UPDATE slices
SET status = ?
WHERE id = ?
RETURNING *;

-- name: UpdateSlicePhase :one
UPDATE slices
SET phase = ?
WHERE id = ?
RETURNING *;

-- name: DeleteSlice :exec
DELETE FROM slices
WHERE id = ?;
