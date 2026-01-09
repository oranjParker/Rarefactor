CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    domain TEXT NOT NULL,
    title TEXT,
    content TEXT NOT NULL,          -- Cleaned text for the search engine
    raw_content_size INT,           -- Auditing crawler efficiency
    crawled_at TIMESTAMPTZ DEFAULT NOW(),

    -- Generic Categorization
    namespace TEXT NOT NULL,        -- e.g., 'finance', 'tech', 'general'
    tags TEXT[],                    -- Searchable array of strings

    -- Linkage
    vector_id UUID UNIQUE,          -- For Qdrant indexing
    metadata JSONB DEFAULT '{}'     -- Catch-all for extra data
);

CREATE INDEX idx_docs_namespace ON documents(namespace);
CREATE INDEX idx_docs_url ON documents(url);