import { useState, useEffect, useCallback } from 'react';
import { Navbar } from './components/Navbar';
import { UploadSection } from './components/UploadSection';
import { DocumentDetails } from './components/DocumentDetails';
import { EventTimeline } from './components/EventTimeline';
import { RecentDocuments } from './components/RecentDocuments';
import type { Document } from './types';

const STORAGE_KEY = 'doc_manager_recent_docs';

export function App() {
  const [documents, setDocuments] = useState<Document[]>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  });

  const [selectedDocument, setSelectedDocument] = useState<Document | null>(() => {
    return documents.length > 0 ? documents[0] : null;
  });

  const [selectedDocId, setSelectedDocId] = useState<string>(() => {
    return documents.length > 0 ? documents[0].id : '';
  });

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(documents));
    } catch (e) {
      console.error('Failed to save documents to localStorage', e);
    }
  }, [documents]);

  const handleDocumentUploaded = (newDoc: Document) => {
    setDocuments((prev) => {
      const filtered = prev.filter((d) => d.id !== newDoc.id);
      return [newDoc, ...filtered];
    });
    setSelectedDocument(newDoc);
    setSelectedDocId(newDoc.id);
  };

  const handleSelectDocument = (doc: Document) => {
    setSelectedDocument(doc);
    setSelectedDocId(doc.id);
  };

  const handleDocumentIdChange = (id: string) => {
    setSelectedDocId(id);
    const existing = documents.find((d) => d.id === id);
    if (existing) {
      setSelectedDocument(existing);
    }
  };

  const handleSnapshotReceived = useCallback((docId: string, updatedFields: Partial<Document>) => {
    if (!docId) return;

    setSelectedDocument((prev) => {
      if (!prev || prev.id !== docId) return prev;
      if (
        prev.status === updatedFields.status &&
        prev.extracted_content === updatedFields.extracted_content &&
        prev.rejection_reason === updatedFields.rejection_reason
      ) {
        return prev;
      }
      return {
        ...prev,
        ...updatedFields,
        status: updatedFields.status || prev.status,
      };
    });

    setDocuments((prevList) => {
      let changed = false;
      const updated = prevList.map((doc) => {
        if (doc.id === docId) {
          if (
            doc.status === updatedFields.status &&
            doc.extracted_content === updatedFields.extracted_content &&
            doc.rejection_reason === updatedFields.rejection_reason
          ) {
            return doc;
          }
          changed = true;
          return {
            ...doc,
            ...updatedFields,
            status: updatedFields.status || doc.status,
          };
        }
        return doc;
      });
      return changed ? updated : prevList;
    });
  }, []);

  const handleClearHistory = () => {
    setDocuments([]);
    localStorage.removeItem(STORAGE_KEY);
  };

  return (
    <div className="app-container">
      <Navbar />

      <main className="main-grid">
        <UploadSection onDocumentUploaded={handleDocumentUploaded} />
        <DocumentDetails document={selectedDocument} />
      </main>

      <section>
        <EventTimeline
          documentId={selectedDocId}
          onDocumentIdChange={handleDocumentIdChange}
          onSnapshotReceived={handleSnapshotReceived}
        />
      </section>

      <section>
        <RecentDocuments
          documents={documents}
          selectedDocumentId={selectedDocId}
          onSelectDocument={handleSelectDocument}
          onClearHistory={handleClearHistory}
        />
      </section>
    </div>
  );
}

export default App;
