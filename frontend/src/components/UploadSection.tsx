import React, { useState, useRef } from 'react';
import type { DragEvent, ChangeEvent } from 'react';
import { UploadCloud, File, AlertTriangle, CheckCircle2, Loader2, Sparkles, X } from 'lucide-react';
import { uploadDocument, ApiError } from '../api';
import type { Document } from '../types';

interface UploadSectionProps {
  onDocumentUploaded: (doc: Document) => void;
}

export const UploadSection: React.FC<UploadSectionProps> = ({ onDocumentUploaded }) => {
  const [file, setFile] = useState<File | null>(null);
  const [typeOfFile, setTypeOfFile] = useState<string>('');
  const [optionalMeta, setOptionalMeta] = useState<string>('');
  const [isDragging, setIsDragging] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [isDuplicate, setIsDuplicate] = useState<boolean>(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (selectedFile: File) => {
    setFile(selectedFile);
    setError(null);
    setIsDuplicate(false);
    setSuccessMessage(null);

    // Auto-detect or infer MIME type
    if (selectedFile.type) {
      setTypeOfFile(selectedFile.type);
    } else {
      const ext = selectedFile.name.split('.').pop()?.toLowerCase();
      if (ext === 'pdf') setTypeOfFile('application/pdf');
      else if (ext === 'csv') setTypeOfFile('text/csv');
      else if (ext === 'json') setTypeOfFile('application/json');
      else if (ext === 'txt') setTypeOfFile('text/plain');
      else setTypeOfFile('application/octet-stream');
    }
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileSelect(e.dataTransfer.files[0]);
    }
  };

  const handleFileInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      handleFileSelect(e.target.files[0]);
    }
  };

  const insertSampleMeta = () => {
    const sample = JSON.stringify(
      {
        department: 'Finance',
        fiscal_year: '2026',
        priority: 'high',
        tag: 'invoices',
      },
      null,
      2
    );
    setOptionalMeta(sample);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) {
      setError('Please choose or drop a file to upload.');
      return;
    }

    setLoading(true);
    setError(null);
    setIsDuplicate(false);
    setSuccessMessage(null);

    try {
      const doc = await uploadDocument(file, typeOfFile, optionalMeta);
      setSuccessMessage(`Document "${doc.name}" accepted successfully! ID: ${doc.id}`);
      onDocumentUploaded(doc);

      // Reset form
      setFile(null);
      setTypeOfFile('');
      setOptionalMeta('');
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } catch (err: any) {
      if (err instanceof ApiError && err.statusCode === 409) {
        setIsDuplicate(true);
        setError(err.message || 'Duplicate document: identical name and content hash already exists.');
      } else {
        setError(err.message || 'Failed to upload document');
      }
    } finally {
      setLoading(false);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="card">
      <div className="card-header">
        <div className="card-title-group">
          <UploadCloud className="card-title-icon" size={20} />
          <h2>Upload &amp; Ingest Document</h2>
        </div>
        <span className="badge badge-neutral">POST /api/documents</span>
      </div>

      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          id="file-input"
          style={{ display: 'none' }}
          onChange={handleFileInputChange}
        />

        {/* Dropzone */}
        <div
          className={`dropzone ${isDragging ? 'active' : ''}`}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
        >
          <div className="dropzone-icon">
            <UploadCloud size={28} />
          </div>
          <div>
            <div className="dropzone-title">Click to upload or drag &amp; drop</div>
            <div className="dropzone-subtitle">Accepts PDF, CSV, Images, JSON, TXT (Any format)</div>
          </div>
        </div>

        {/* File Preview */}
        {file && (
          <div className="selected-file-preview animate-fade-in">
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <File size={18} color="#818cf8" />
              <div>
                <strong style={{ display: 'block', color: '#f8fafc' }}>{file.name}</strong>
                <span style={{ fontSize: '0.78rem', color: '#94a3b8' }}>
                  {formatFileSize(file.size)} • {file.type || 'Custom Type'}
                </span>
              </div>
            </div>
            <button
              type="button"
              className="copy-button"
              onClick={(e) => {
                e.stopPropagation();
                setFile(null);
                if (fileInputRef.current) fileInputRef.current.value = '';
              }}
              title="Remove file"
            >
              <X size={18} />
            </button>
          </div>
        )}

        {/* File Type */}
        <div className="form-group">
          <label className="form-label" htmlFor="typeOfFile">
            <span>MIME Type / Format</span>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Optional</span>
          </label>
          <input
            id="typeOfFile"
            type="text"
            className="form-input"
            placeholder="e.g. application/pdf, text/csv"
            value={typeOfFile}
            onChange={(e) => setTypeOfFile(e.target.value)}
          />
        </div>

        {/* Optional Metadata */}
        <div className="form-group">
          <div className="form-label">
            <span>Optional Metadata (JSON or text)</span>
            <button
              type="button"
              className="copy-button"
              onClick={insertSampleMeta}
              style={{ fontSize: '0.75rem', display: 'flex', alignItems: 'center', gap: '4px' }}
            >
              <Sparkles size={12} color="#06b6d4" />
              <span>Fill Sample JSON</span>
            </button>
          </div>
          <textarea
            id="optionalMeta"
            className="form-textarea"
            placeholder='e.g. {"department": "Finance", "tag": "invoice"}'
            value={optionalMeta}
            onChange={(e) => setOptionalMeta(e.target.value)}
            rows={3}
          />
        </div>

        {/* Duplicate Warning Alert */}
        {isDuplicate && (
          <div className="alert alert-error animate-fade-in" id="duplicate-alert">
            <AlertTriangle size={20} style={{ flexShrink: 0, marginTop: '2px' }} />
            <div>
              <strong>Duplicate Document Detected (409 Conflict):</strong>
              <div style={{ marginTop: '4px', fontSize: '0.82rem' }}>
                A document with identical name and content hash already exists in PostgreSQL database. Duplicate uploads are rejected to ensure idempotency.
              </div>
            </div>
          </div>
        )}

        {/* Generic Error Alert */}
        {error && !isDuplicate && (
          <div className="alert alert-error animate-fade-in">
            <AlertTriangle size={18} style={{ flexShrink: 0 }} />
            <span>{error}</span>
          </div>
        )}

        {/* Success Alert */}
        {successMessage && (
          <div className="alert alert-success animate-fade-in">
            <CheckCircle2 size={18} style={{ flexShrink: 0 }} />
            <span>{successMessage}</span>
          </div>
        )}

        <button
          type="submit"
          className="btn btn-primary"
          disabled={!file || loading}
          id="submit-upload-btn"
        >
          {loading ? (
            <>
              <Loader2 size={18} className="animate-spin" />
              <span>Uploading &amp; Registering...</span>
            </>
          ) : (
            <>
              <UploadCloud size={18} />
              <span>Upload Document</span>
            </>
          )}
        </button>
      </form>
    </div>
  );
};
