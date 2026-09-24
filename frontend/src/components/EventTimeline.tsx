import React, { useState, useEffect, useCallback, useRef } from 'react';
import { History, RefreshCw, Radio, CheckCircle2, AlertOctagon, Clock, Search, ChevronDown, ChevronRight } from 'lucide-react';
import { getDocumentEvents, ApiError } from '../api';
import type { DocumentSnapshot, Document } from '../types';

interface EventTimelineProps {
  documentId: string;
  onDocumentIdChange: (id: string) => void;
  onSnapshotReceived?: (docId: string, doc: Partial<Document>) => void;
}

export const EventTimeline: React.FC<EventTimelineProps> = ({
  documentId,
  onDocumentIdChange,
  onSnapshotReceived,
}) => {
  const [events, setEvents] = useState<DocumentSnapshot[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [autoPoll, setAutoPoll] = useState<boolean>(true);
  const [expandedIndices, setExpandedIndices] = useState<Record<number, boolean>>({});
  const [searchId, setSearchId] = useState<string>(documentId);

  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const onSnapshotReceivedRef = useRef(onSnapshotReceived);

  useEffect(() => {
    onSnapshotReceivedRef.current = onSnapshotReceived;
  });

  useEffect(() => {
    setSearchId(documentId);
  }, [documentId]);

  const fetchEvents = useCallback(
    async (idToQuery: string, isPolling = false) => {
      if (!idToQuery.trim()) return;

      if (!isPolling) setLoading(true);
      setError(null);

      try {
        const fetchedEvents = await getDocumentEvents(idToQuery);
        // Sort descending by timestamp if not already sorted
        fetchedEvents.sort(
          (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
        );
        setEvents(fetchedEvents);

        // Check latest event status
        if (fetchedEvents.length > 0) {
          const latest = fetchedEvents[0];
          try {
            const parsed = JSON.parse(latest.db_snapshot);
            if (onSnapshotReceivedRef.current) {
              onSnapshotReceivedRef.current(idToQuery, parsed);
            }
          } catch {
            // ignore
          }

          // If reached terminal status, turn off auto-poll
          const latestStatus = latest.status.toLowerCase();
          if (latestStatus === 'success' || latestStatus.includes('failed') || latestStatus === 'expired') {
            setAutoPoll(false);
          }
        }
      } catch (err: any) {
        if (err instanceof ApiError && err.statusCode === 400) {
          setError(err.message || 'Invalid Document UUID format.');
        } else {
          setError(err.message || 'Failed to query Cassandra document snapshots.');
        }
      } finally {
        if (!isPolling) setLoading(false);
      }
    },
    []
  );

  // Trigger query whenever documentId changes
  useEffect(() => {
    if (documentId.trim()) {
      fetchEvents(documentId, false);
      setAutoPoll(true);
    } else {
      setEvents([]);
    }
  }, [documentId, fetchEvents]);

  // Polling loop with 10-second interval
  useEffect(() => {
    if (!autoPoll || !documentId.trim()) {
      if (pollTimerRef.current) clearInterval(pollTimerRef.current);
      return;
    }

    pollTimerRef.current = setInterval(() => {
      fetchEvents(documentId, true);
    }, 10000);

    return () => {
      if (pollTimerRef.current) clearInterval(pollTimerRef.current);
    };
  }, [autoPoll, documentId, fetchEvents]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchId.trim()) {
      onDocumentIdChange(searchId.trim());
    }
  };

  const toggleExpand = (index: number) => {
    setExpandedIndices((prev) => ({
      ...prev,
      [index]: !prev[index],
    }));
  };

  const getNodeClass = (status: string) => {
    const s = status.toLowerCase();
    if (s === 'success') return 'timeline-node success';
    if (s === 'failed' || s.includes('failed')) return 'timeline-node failed';
    if (s === 'processing') return 'timeline-node processing';
    return 'timeline-node';
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

  return (
    <div className="card">
      <div className="card-header">
        <div className="card-title-group">
          <History className="card-title-icon" size={20} />
          <h2>Cassandra Event History &amp; Audit Trail</h2>
        </div>
        <span className="badge badge-neutral">GET /api/documents/:id/events</span>
      </div>

      {/* UUID Search Form */}
      <form onSubmit={handleSearchSubmit} style={{ display: 'flex', gap: '8px' }}>
        <input
          type="text"
          className="form-input"
          placeholder="Lookup by Document UUID (e.g. 550e8400-e29b-41d4-a716-446655440000)"
          value={searchId}
          onChange={(e) => setSearchId(e.target.value)}
          id="search-uuid-input"
        />
        <button type="submit" className="btn btn-secondary btn-sm" title="Lookup events">
          <Search size={16} />
        </button>
      </form>

      {/* Controls Bar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '10px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={autoPoll}
              onChange={(e) => setAutoPoll(e.target.checked)}
              disabled={!documentId}
            />
            <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              <Radio size={14} color={autoPoll ? '#38bdf8' : 'var(--text-muted)'} />
              <span>Live Auto-Poll (10s)</span>
            </span>
          </label>
        </div>

        <button
          type="button"
          className="btn btn-secondary btn-sm"
          onClick={() => fetchEvents(documentId, false)}
          disabled={!documentId || loading}
          id="refresh-events-btn"
        >
          <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          <span>Refresh</span>
        </button>
      </div>

      {/* Error Alert */}
      {error && (
        <div className="alert alert-error animate-fade-in">
          <AlertOctagon size={18} style={{ flexShrink: 0 }} />
          <span>{error}</span>
        </div>
      )}

      {/* Timeline List */}
      {!documentId ? (
        <div style={{ padding: '30px 10px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '0.88rem' }}>
          Enter a Document UUID or select a document above to inspect its event history in Cassandra.
        </div>
      ) : events.length === 0 && !loading ? (
        <div style={{ padding: '30px 10px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '0.88rem' }}>
          No event snapshots found in Cassandra for this document ID.
        </div>
      ) : (
        <div className="timeline">
          {events.map((ev, idx) => {
            const isExpanded = !!expandedIndices[idx];
            let parsedSnapshot: any = null;
            try {
              parsedSnapshot = JSON.parse(ev.db_snapshot);
            } catch {
              parsedSnapshot = ev.db_snapshot;
            }

            return (
              <div key={`${ev.timestamp}-${idx}`} className="timeline-item animate-fade-in">
                <div className={getNodeClass(ev.status)} />
                <div className="timeline-content">
                  <div className="timeline-header">
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      {getStatusBadge(ev.status)}
                    </div>
                    <span className="timeline-time">
                      {new Date(ev.timestamp).toLocaleTimeString()} • {new Date(ev.timestamp).toLocaleDateString()}
                    </span>
                  </div>

                  {/* Quick summary line if extracted content or rejection exists */}
                  {parsedSnapshot && typeof parsedSnapshot === 'object' && (
                    <div style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
                      {parsedSnapshot.extracted_content && (
                        <div style={{ color: '#34d399', fontSize: '0.8rem', marginTop: '2px' }}>
                          ✓ Extracted text captured in snapshot
                        </div>
                      )}
                      {parsedSnapshot.rejection_reason && (
                        <div style={{ color: '#fb7185', fontSize: '0.8rem', marginTop: '2px' }}>
                          ✗ Reason: {parsedSnapshot.rejection_reason}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Collapsible raw snapshot */}
                  <div style={{ marginTop: '4px' }}>
                    <button
                      type="button"
                      className="copy-button"
                      onClick={() => toggleExpand(idx)}
                      style={{ fontSize: '0.78rem', color: 'var(--text-muted)' }}
                    >
                      {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                      <span>{isExpanded ? 'Hide DB Snapshot' : 'View DB Snapshot'}</span>
                    </button>

                    {isExpanded && (
                      <pre className="json-viewer animate-fade-in" style={{ marginTop: '8px' }}>
                        {typeof parsedSnapshot === 'object'
                          ? JSON.stringify(parsedSnapshot, null, 2)
                          : String(parsedSnapshot)}
                      </pre>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
