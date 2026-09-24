import React, { useEffect, useState } from 'react';
import { FileText, ExternalLink, Activity, RefreshCw } from 'lucide-react';
import { checkBackendConnection } from '../api';

export const Navbar: React.FC = () => {
  const [isOnline, setIsOnline] = useState<boolean | null>(null);
  const [checking, setChecking] = useState<boolean>(false);

  const pingBackend = async () => {
    setChecking(true);
    const online = await checkBackendConnection();
    setIsOnline(online);
    setChecking(false);
  };

  useEffect(() => {
    pingBackend();
    const interval = setInterval(pingBackend, 15000);
    return () => clearInterval(interval);
  }, []);

  return (
    <header className="app-header">
      <div className="header-brand">
        <div className="header-logo">
          <FileText size={24} />
        </div>
        <div className="header-title-wrap">
          <h1>DocManager</h1>
          <p>Document Ingestion &amp; Cassandra Event Audit Service</p>
        </div>
      </div>

      <div className="header-actions">
        <div 
          className="status-indicator" 
          title="Backend API Connectivity Status"
          style={{ cursor: 'pointer' }}
          onClick={pingBackend}
        >
          <span className={`status-dot ${isOnline ? 'online' : 'offline'}`} />
          <span>
            {checking
              ? 'Checking...'
              : isOnline
              ? 'Backend API Online'
              : 'Backend Offline'}
          </span>
          <RefreshCw size={12} className={checking ? 'animate-spin' : ''} />
        </div>

        <a
          href="/swagger/index.html"
          target="_blank"
          rel="noopener noreferrer"
          className="header-link-btn"
          id="swagger-docs-btn"
        >
          <Activity size={15} />
          <span>Swagger Docs</span>
          <ExternalLink size={13} />
        </a>
      </div>
    </header>
  );
};
