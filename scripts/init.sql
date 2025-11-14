-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on owner_id for efficient queries
CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON documents(owner_id);

-- Insert sample data
INSERT INTO documents (id, title, content, owner_id, created_at, updated_at) VALUES
    ('doc-1', 'サンプルドキュメント1', 'これはuser-1が作成したドキュメントです', 'user-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('doc-2', 'サンプルドキュメント2', 'これはuser-2が作成したドキュメントです', 'user-2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('doc-3', 'サンプルドキュメント3', 'これはuser-1が作成した別のドキュメントです', 'user-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;
