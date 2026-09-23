CREATE TABLE IF NOT EXISTS documents (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255),
	path VARCHAR(255),
	size BIGINT,
	mimeType VARCHAR(50),
	status VARCHAR(50),
	rejection_reason TEXT,
	extracted_content JSONB,
	optional_meta TEXT,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
