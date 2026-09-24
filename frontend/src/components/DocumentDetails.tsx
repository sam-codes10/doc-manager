import React, { useState } from 'react';
import type { Document } from '../types';
import { FileText, Copy, Check, Hash, HardDrive, AlertOctagon, CheckCircle2, Clock } from 'lucide-react';

interface DocumentDetailsProps {
  document: Document | null;
}

export const DocumentDetails: React.FC<DocumentDetailsProps> = ({ document }) => {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  if (!document) {
    return (
      <div className="card" style={{ height: '100%', justifyContent: 'center', alignItems: 'center', textAlign: 'center', minHeight: '340px' }}>
        <FileText size={44} style={{ color: 'var(--text-muted)', opacity: 0.5, marginBottom: '12px' }} />
        <h3 style={{ color: 'var(--text-secondary)', fontSize: '1rem', fontWeight: 600 }}>No Document Selected</h3>
        <p style={{ color: 'var(--text-muted)', fontSize: '0.82rem', maxWidth: '320px', marginTop: '6px' }}>
          Upload a new document or pick one from the recent documents list below to inspect its record and audit events.
        </p>
      </div>
    );
  }

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const getStatusBadge = (status: string) => {
    const s = status.toLowerCase();
    if (s === 'success') {
      return (
        <span className="badge badge-success">
          <CheckCircle2 size={12} />
          {status}
        </span>
      );
    }
    if (s === 'failed' || s.includes('failed')) {
      return (
        <span className="badge badge-error">
          <AlertOctagon size={12} />
          {status}
        </span>
      );
    }
    if (s === 'processing') {
      return (
        <span className="badge badge-warning animate-pulse">
          <Clock size={12} />
          {status}
        </span>
      );
    }
    return (
      <span className="badge badge-info">
        <Clock size={12} />
        {status}
      </span>
    );
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  let formattedExtractedContent = document.extracted_content;
  try {
    if (document.extracted_content) {
      const parsed = JSON.parse(document.extracted_content);
      formattedExtractedContent = JSON.stringify(parsed, null, 2);
    }
  } catch {
    // Keep as string
  }

  return (
    <div className="card animate-fade-in">
      <div className="card-header">
        <div className="card-title-group">
          <FileText className="card-title-icon" size={20} />
          <h2>Document Record</h2>
        </div>
        <div>{getStatusBadge(document.status)}</div>
      </div>

      <table className="data-table">
        <tbody>
          <tr>
            <th>Document ID</th>
            <td>
              <span className="code-snippet">{document.id}</span>
              <button
                type="button"
                className="copy-button"
                onClick={() => copyToClipboard(document.id, 'id')}
                title="Copy Document UUID"
              >
                {copiedKey === 'id' ? <Check size={14} color="#34d399" /> : <Copy size={14} />}
              </button>
            </td>
          </tr>
          <tr>
            <th>File Name</th>
            <td>
              <strong>{document.name}</strong>
            </td>
          </tr>
          <tr>
            <th>Content Hash</th>
            <td>
              <span className="code-snippet" title={document.content_hash}>
                {document.content_hash.slice(0, 16)}...
              </span>
              <button
                type="button"
                className="copy-button"
                onClick={() => copyToClipboard(document.content_hash, 'hash')}
                title="Copy full SHA-256 hash"
              >
                {copiedKey === 'hash' ? <Check size={14} color="#34d399" /> : <Hash size={14} />}
              </button>
            </td>
          </tr>
          <tr>
            <th>Size &amp; MIME Type</th>
            <td>
              {formatFileSize(document.size)} • {document.mime_type}
            </td>
          </tr>
          <tr>
            <th>Storage Path (S3)</th>
            <td>
              <span className="code-snippet" style={{ wordBreak: 'break-all', fontSize: '0.78rem' }}>
                {document.path || 'Not stored'}
              </span>
              {document.path && (
                <button
                  type="button"
                  className="copy-button"
                  onClick={() => copyToClipboard(document.path, 'path')}
                  title="Copy S3 Path"
                >
                  {copiedKey === 'path' ? <Check size={14} color="#34d399" /> : <HardDrive size={14} />}
                </button>
              )}
            </td>
          </tr>
          <tr>
            <th>Created At</th>
            <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
              {new Date(document.created_at).toLocaleString()}
            </td>
          </tr>
          {document.updated_at && (
            <tr>
              <th>Updated At</th>
              <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                {new Date(document.updated_at).toLocaleString()}
              </td>
            </tr>
          )}
        </tbody>
      </table>

      {/* Extracted Content (OCR / Processed Result) */}
      {formattedExtractedContent && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <span style={{ fontSize: '0.82rem', fontWeight: 600, color: '#34d399' }}>
            ✓ Extracted Content
          </span>
          <pre className="json-viewer">{formattedExtractedContent}</pre>
        </div>
      )}

      {/* Rejection Reason (Failed Status) */}
      {document.rejection_reason && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <span style={{ fontSize: '0.82rem', fontWeight: 600, color: '#fb7185' }}>
            ✗ Rejection / Error Reason
          </span>
          <div className="alert alert-error" style={{ fontSize: '0.82rem' }}>
            {document.rejection_reason}
          </div>
        </div>
      )}

      {/* Optional Metadata */}
      {document.optional_meta && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <span style={{ fontSize: '0.82rem', fontWeight: 600, color: 'var(--text-muted)' }}>
            User Metadata
          </span>
          <pre className="json-viewer" style={{ color: '#cbd5e1' }}>
            {document.optional_meta}
          </pre>
        </div>
      )}
    </div>
  );
};
