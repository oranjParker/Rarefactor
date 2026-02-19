CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    parent_id TEXT,
    namespace TEXT NOT NULL,
    domain TEXT NOT NULL,
    source TEXT NOT NULL,

    title TEXT,
    summary TEXT,
    content TEXT NOT NULL,
    cleaned_content TEXT,
    content_hash TEXT,
    crawled_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),

    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_docs_namespace ON documents(namespace);
CREATE INDEX IF NOT EXISTS idx_docs_domain ON documents(domain);
CREATE INDEX IF NOT EXISTS idx_docs_parent_id ON documents(parent_id);
CREATE INDEX IF NOT EXISTS idx_docs_last_seen ON documents(last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_docs_content_hash ON documents(content_hash);

CREATE INDEX IF NOT EXISTS idx_docs_metadata ON documents USING GIN (metadata);