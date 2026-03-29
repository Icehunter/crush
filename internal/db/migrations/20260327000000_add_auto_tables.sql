-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS milestones (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    phase TEXT NOT NULL DEFAULT 'pre_planning',
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE TRIGGER IF NOT EXISTS update_milestones_updated_at
AFTER UPDATE ON milestones
BEGIN
UPDATE milestones SET updated_at = strftime('%s', 'now')
WHERE id = new.id;
END;

CREATE TABLE IF NOT EXISTS slices (
    id TEXT PRIMARY KEY,
    milestone_id TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    phase TEXT NOT NULL DEFAULT 'pre_planning',
    sort_order INTEGER NOT NULL DEFAULT 0,
    depends_on TEXT,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (milestone_id) REFERENCES milestones (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_slices_milestone_id ON slices (milestone_id);

CREATE TRIGGER IF NOT EXISTS update_slices_updated_at
AFTER UPDATE ON slices
BEGIN
UPDATE slices SET updated_at = strftime('%s', 'now')
WHERE id = new.id;
END;

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    slice_id TEXT NOT NULL,
    milestone_id TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    phase TEXT NOT NULL DEFAULT 'pre_planning',
    sort_order INTEGER NOT NULL DEFAULT 0,
    description TEXT,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (slice_id) REFERENCES slices (id) ON DELETE CASCADE,
    FOREIGN KEY (milestone_id) REFERENCES milestones (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_slice_id ON tasks (slice_id);
CREATE INDEX IF NOT EXISTS idx_tasks_milestone_id ON tasks (milestone_id);

CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
AFTER UPDATE ON tasks
BEGIN
UPDATE tasks SET updated_at = strftime('%s', 'now')
WHERE id = new.id;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_tasks_updated_at;
DROP INDEX IF EXISTS idx_tasks_milestone_id;
DROP INDEX IF EXISTS idx_tasks_slice_id;
DROP TABLE IF EXISTS tasks;

DROP TRIGGER IF EXISTS update_slices_updated_at;
DROP INDEX IF EXISTS idx_slices_milestone_id;
DROP TABLE IF EXISTS slices;

DROP TRIGGER IF EXISTS update_milestones_updated_at;
DROP TABLE IF EXISTS milestones;
-- +goose StatementEnd
