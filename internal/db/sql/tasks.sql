-- name: CreateTask :one
INSERT INTO tasks (
    id,
    slice_id,
    milestone_id,
    title,
    status,
    phase,
    sort_order,
    description,
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
    ?,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ? LIMIT 1;

-- name: ListTasksBySlice :many
SELECT * FROM tasks
WHERE slice_id = ?
ORDER BY sort_order;

-- name: ListTasksByMilestone :many
SELECT * FROM tasks
WHERE milestone_id = ?
ORDER BY sort_order;

-- name: UpdateTaskStatus :one
UPDATE tasks
SET status = ?
WHERE id = ?
RETURNING *;

-- name: UpdateTaskPhase :one
UPDATE tasks
SET phase = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = ?;
