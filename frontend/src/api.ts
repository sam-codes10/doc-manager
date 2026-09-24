import type { ApiRes, Document, DocumentSnapshot } from './types';

export class ApiError extends Error {
  statusCode: number;
  data?: any;

  constructor(message: string, statusCode: number, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.statusCode = statusCode;
    this.data = data;
  }
}

/**
 * Upload a document file with optional format and metadata.
 * Uses relative path `/api/documents` so Vite development proxy forwards to backend,
 * completely avoiding CORS issues.
 */
export async function uploadDocument(
  file: File,
  typeOfFile?: string,
  optionalMeta?: string
): Promise<Document> {
  const formData = new FormData();
  formData.append('file', file);
  if (typeOfFile && typeOfFile.trim()) {
    formData.append('typeOfFile', typeOfFile.trim());
  }
  if (optionalMeta && optionalMeta.trim()) {
    formData.append('optionalMeta', optionalMeta.trim());
  }

  const response = await fetch('/api/documents', {
    method: 'POST',
    body: formData,
  });

  const body: ApiRes<Document> = await response.json().catch(() => ({
    status: false,
    message: response.statusText || 'Unknown server response',
    data: null as any,
  }));

  if (!response.ok || !body.status) {
    let msg = body.message || `Upload failed with HTTP status ${response.status}`;
    if (response.status === 409) {
      msg = body.message || 'Duplicate document: a file with the same name and content already exists.';
    }
    throw new ApiError(msg, response.status, body);
  }

  return body.data;
}

/**
 * Query Cassandra event timeline for a given document UUID.
 */
export async function getDocumentEvents(documentId: string): Promise<DocumentSnapshot[]> {
  const trimmedId = documentId.trim();
  if (!trimmedId) {
    throw new ApiError('Document UUID is required', 400);
  }

  const response = await fetch(`/api/documents/${encodeURIComponent(trimmedId)}/events`);
  const body: ApiRes<DocumentSnapshot[]> = await response.json().catch(() => ({
    status: false,
    message: response.statusText || 'Unknown server response',
    data: [],
  }));

  if (!response.ok || !body.status) {
    throw new ApiError(
      body.message || `Failed to fetch events (HTTP ${response.status})`,
      response.status,
      body
    );
  }

  return body.data || [];
}

/**
 * Check backend connection by pinging Swagger doc or root
 */
export async function checkBackendConnection(): Promise<boolean> {
  try {
    const res = await fetch('/swagger/doc.json', { method: 'GET' });
    return res.ok;
  } catch {
    return false;
  }
}
