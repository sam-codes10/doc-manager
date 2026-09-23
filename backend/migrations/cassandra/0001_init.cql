-- Keyspace setup for doc-manager
CREATE KEYSPACE IF NOT EXISTS doc_manager 
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE doc_manager;

-- Document snapshots table to track status transitions and full DB snapshots over time
-- In Cassandra CQL, JSON/JSONB data is stored using the TEXT data type
CREATE TABLE IF NOT EXISTS document_snapshots (
    document_id UUID,
    status TEXT,
    db_snapshot TEXT, -- JSON snapshot of postgres documents row
    timestamp TIMESTAMP,
    PRIMARY KEY (document_id, timestamp)
) WITH CLUSTERING ORDER BY (timestamp DESC);
