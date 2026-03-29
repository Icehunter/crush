-- name: CreateMilestone :one
INSERT INTO milestones (
    id,
    title,
    status,
    phase,
    created_at,
    updated_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING *;

-- name: GetMilestone :one
SELECT * FROM milestones
WHERE id = ? LIMIT 1;

-- name: ListMilestones :many
SELECT * FROM milestones
ORDER BY created_at DESC;

-- name: UpdateMilestoneStatus :one
UPDATE milestones
SET status = ?
WHERE id = ?
RETURNING *;

-- name: UpdateMilestonePhase :one
UPDATE milestones
SET phase = ?
WHERE id = ?
RETURNING *;

-- name: DeleteMilestone :exec
DELETE FROM milestones
WHERE id = ?;
