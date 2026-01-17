CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    domain TEXT NOT NULL,
    title TEXT,
    content TEXT NOT NULL,
    raw_content_size INT,
    crawled_at TIMESTAMPTZ DEFAULT NOW(),

    namespace TEXT NOT NULL,
    tags TEXT[],

    vector_id UUID UNIQUE,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_docs_namespace ON documents(namespace);
CREATE INDEX IF NOT EXISTS idx_docs_url ON documents(url);
CREATE INDEX IF NOT EXISTS idx_docs_domain ON documents(domain);
CREATE INDEX IF NOT EXISTS idx_docs_crawled_at ON documents(crawled_at DESC);