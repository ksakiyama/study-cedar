-- Create user_groups table
CREATE TABLE IF NOT EXISTS user_groups (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create document_groups table
CREATE TABLE IF NOT EXISTS document_groups (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    document_group_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (document_group_id) REFERENCES document_groups(id)
);

-- Create group_associations table (N:N relationship between document_groups and user_groups)
CREATE TABLE IF NOT EXISTS group_associations (
    id SERIAL PRIMARY KEY,
    document_group_id VARCHAR(255) NOT NULL,
    user_group_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (document_group_id) REFERENCES document_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups(id) ON DELETE CASCADE,
    UNIQUE(document_group_id, user_group_id)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON documents(owner_id);
CREATE INDEX IF NOT EXISTS idx_documents_group_id ON documents(document_group_id);
CREATE INDEX IF NOT EXISTS idx_group_associations_doc_group ON group_associations(document_group_id);
CREATE INDEX IF NOT EXISTS idx_group_associations_user_group ON group_associations(user_group_id);

-- Insert sample user groups
INSERT INTO user_groups (id, name, created_at) VALUES
    ('user-group-engineering', 'Engineering Team', CURRENT_TIMESTAMP),
    ('user-group-sales', 'Sales Team', CURRENT_TIMESTAMP),
    ('user-group-management', 'Management', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- Insert sample document groups
INSERT INTO document_groups (id, name, created_at) VALUES
    ('doc-group-technical', 'Technical Documentation', CURRENT_TIMESTAMP),
    ('doc-group-sales', 'Sales Materials', CURRENT_TIMESTAMP),
    ('doc-group-internal', 'Internal Documents', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- Insert sample documents
INSERT INTO documents (id, title, content, owner_id, document_group_id, created_at, updated_at) VALUES
    ('doc-1', 'Technical Specification', 'This is a technical specification document created by user-1', 'user-1', 'doc-group-technical', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('doc-2', 'Sales Proposal', 'This is a sales proposal document created by user-2', 'user-2', 'doc-group-sales', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('doc-3', 'Internal Memo', 'This is an internal memo created by user-1', 'user-1', 'doc-group-internal', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('doc-4', 'API Documentation', 'API documentation for engineers', 'user-1', 'doc-group-technical', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('doc-5', 'Quarterly Report', 'Management quarterly report', 'user-3', 'doc-group-internal', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- Insert sample group associations
-- Engineering team can access technical documentation
INSERT INTO group_associations (document_group_id, user_group_id, created_at) VALUES
    ('doc-group-technical', 'user-group-engineering', CURRENT_TIMESTAMP),
    ('doc-group-sales', 'user-group-sales', CURRENT_TIMESTAMP),
    ('doc-group-internal', 'user-group-management', CURRENT_TIMESTAMP),
    ('doc-group-internal', 'user-group-engineering', CURRENT_TIMESTAMP)
ON CONFLICT (document_group_id, user_group_id) DO NOTHING;
