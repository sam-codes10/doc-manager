CREATE TABLE IF NOT EXISTS documents (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL,
	content_hash VARCHAR(64),
	path VARCHAR(255),
	size BIGINT,
	mimeType VARCHAR(50),
	status VARCHAR(50),
	rejection_reason TEXT,
	extracted_content JSONB,
	optional_meta TEXT,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT uq_document_content_hash_name UNIQUE (content_hash, name)
);

-- Idempotent column and constraint addition for existing databases
ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64);
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_document_content_hash_name'
    ) THEN
        ALTER TABLE documents ADD CONSTRAINT uq_document_content_hash_name UNIQUE (content_hash, name);
    END IF;
END $$;

