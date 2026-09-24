import React, { useState } from 'react';
import type { Document } from '../types';
import { Files, Trash2, CheckCircle2, AlertOctagon, Clock } from 'lucide-react';

interface RecentDocumentsProps {
  documents: Document[];
  selectedDocumentId: string | null;
  onSelectDocument: (doc: Document) => void;
  onClearHistory: () => void;
}

export const RecentDocuments: React.FC<RecentDocumentsProps> = ({
  documents,
  selectedDocumentId,
  onSelectDocument,
  onClearHistory,
}) => {
  const [filter, setFilter] = useState<string>('all');

  const filteredDocs = documents.filter((doc) => {
    if (filter === 'all') return true;
    return doc.status.toLowerCase().includes(filter.toLowerCase());
  });

  const getStatusBadge = (status: string) => {
    const s = status.toLowerCase();
    if (s === 'success') {
      return (
        <span className="badge badge-success">
          <CheckCircle2 size={11} />
          {status}
        </span>
      );
    }
    if (s === 'failed' || s.includes('failed')) {
      return (
        <span className="badge badge-error">
          <AlertOctagon size={11} />
          {status}
        </span>
      );
    }
    if (s === 'processing') {
      return (
        <span className="badge badge-warning animate-pulse">
          <Clock size={11} />
          {status}
        </span>
      );
    }
    return (
      <span className="badge badge-info">
        <Clock size={11} />
        {status}
      </span>
    );
  };

  if (documents.length === 0) {
    return null;
  }

  return (
    <div className="card">
      <div className="card-header">
        <div className="card-title-group">
          <Files className="card-title-icon" size={20} />
          <h2>Recent Documents Session History</h2>
          <span className="badge badge-neutral" style={{ marginLeft: '6px' }}>
            {documents.length}
          </span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          {/* Quick Filters */}
          <div style={{ display: 'flex', gap: '4px' }}>
            {['all', 'success', 'processing', 'failed'].map((st) => (
              <button
                key={st}
                type="button"
                className={`btn btn-secondary btn-sm ${filter === st ? 'btn-primary' : ''}`}
                style={{ textTransform: 'capitalize', fontSize: '0.75rem', padding: '4px 8px' }}
                onClick={() => setFilter(st)}
              >
                {st}
              </button>
            ))}
          </div>

          <button
            type="button"
            className="copy-button"
            title="Clear saved local history"
            onClick={onClearHistory}
            style={{ color: 'var(--text-muted)' }}
          >
            <Trash2 size={16} />
          </button>
        </div>
      </div>

      <div className="history-list">
        {filteredDocs.map((doc) => {
          const isSelected = selectedDocumentId === doc.id;
          return (
            <div
              key={doc.id}
              className={`history-card ${isSelected ? 'active' : ''}`}
              onClick={() => onSelectDocument(doc)}
            >
              <div className="history-card-header">
                <span className="history-card-title" title={doc.name}>
                  {doc.name}
                </span>
                {getStatusBadge(doc.status)}
              </div>
              <div className="history-card-meta">
                <span className="code-snippet" style={{ fontSize: '0.72rem' }}>
                  {doc.id.slice(0, 8)}...
                </span>
                <span>{new Date(doc.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
