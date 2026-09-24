export interface Document {
  id: string;
  name: string;
  content_hash: string;
  path: string;
  size: number;
  mime_type: string;
  status: 'uploaded' | 'processing' | 'success' | 'failed' | 'expired' | 's3 uploaded' | 's3 uploaded failed' | string;
  rejection_reason?: string | null;
  extracted_content?: string | null;
  optional_meta?: string | null;
  created_at: string;
  updated_at: string;
}

export interface DocumentSnapshot {
  document_id: string;
  status: string;
  db_snapshot: string;
  timestamp: string;
}

export interface ApiRes<T> {
  status: boolean;
  message: string;
  data: T;
}

export interface UploadFormData {
  file: File | null;
  typeOfFile: string;
  optionalMeta: string;
}
