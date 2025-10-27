package datamonkey

// Migration represents a database schema migration
type Migration struct {
	Version int
	Name    string
	Up      string // SQL to apply the migration
	Down    string // SQL to rollback the migration
}

// GetMigrations returns all migrations in order
func GetMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "initial_unified_schema",
			Up: `
-- ============================================================================
-- SESSIONS TABLE
-- Tracks user sessions and authentication
-- ============================================================================
CREATE TABLE IF NOT EXISTS sessions (
    subject TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    last_seen INTEGER NOT NULL,
    metadata TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_last_seen ON sessions(last_seen);

-- ============================================================================
-- DATASETS TABLE
-- Tracks uploaded datasets (alignments, trees, etc.)
-- ============================================================================
CREATE TABLE IF NOT EXISTS datasets (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    metadata_name TEXT NOT NULL,
    metadata_description TEXT,
    metadata_type TEXT NOT NULL,
    metadata_created INTEGER NOT NULL,
    metadata_updated INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    data_json TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES sessions(subject) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_datasets_user_id ON datasets(user_id);
CREATE INDEX IF NOT EXISTS idx_datasets_type ON datasets(metadata_type);
CREATE INDEX IF NOT EXISTS idx_datasets_created ON datasets(metadata_created);
CREATE INDEX IF NOT EXISTS idx_datasets_content_hash ON datasets(content_hash);

-- ============================================================================
-- JOBS TABLE
-- Tracks analysis jobs submitted to the scheduler
-- ============================================================================
CREATE TABLE IF NOT EXISTS jobs (
    job_id TEXT PRIMARY KEY,
    scheduler_job_id TEXT NOT NULL,
    user_id TEXT,
    alignment_id TEXT,
    tree_id TEXT,
    method_type TEXT,
    status TEXT DEFAULT 'pending',
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (user_id) REFERENCES sessions(subject) ON DELETE CASCADE,
    FOREIGN KEY (alignment_id) REFERENCES datasets(id) ON DELETE CASCADE,
    FOREIGN KEY (tree_id) REFERENCES datasets(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_jobs_scheduler_job_id ON jobs(scheduler_job_id);
CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_alignment_id ON jobs(alignment_id);
CREATE INDEX IF NOT EXISTS idx_jobs_tree_id ON jobs(tree_id);
CREATE INDEX IF NOT EXISTS idx_jobs_method_type ON jobs(method_type);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);

-- ============================================================================
-- CONVERSATIONS TABLE
-- Tracks chat conversations with AI
-- ============================================================================
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    title TEXT,
    created INTEGER NOT NULL,
    updated INTEGER NOT NULL,
    FOREIGN KEY (subject) REFERENCES sessions(subject) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_conversations_subject ON conversations(subject);
CREATE INDEX IF NOT EXISTS idx_conversations_created ON conversations(created);

-- ============================================================================
-- MESSAGES TABLE
-- Stores individual messages within conversations
-- ============================================================================
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
`,
			Down: `
-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS datasets;
DROP TABLE IF EXISTS sessions;
`,
		},
	}
}
