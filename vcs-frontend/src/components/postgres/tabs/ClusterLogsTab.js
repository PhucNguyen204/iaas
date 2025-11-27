import React, { useState, useEffect } from 'react';
import { FileText, RefreshCw, Download, Search } from 'lucide-react';
import { clusterAPI } from '../../../api';

const ClusterLogsTab = ({ clusterId }) => {
  const [logs, setLogs] = useState([]);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [filter, setFilter] = useState('');

  useEffect(() => {
    loadLogs();
    if (autoRefresh) {
      const interval = setInterval(loadLogs, 5000);
      return () => clearInterval(interval);
    }
  }, [clusterId, autoRefresh]);

  const loadLogs = async () => {
    try {
      const response = await clusterAPI.getLogs(clusterId, 100);
      setLogs(response.data.data?.logs || []);
    } catch (error) {
      console.error('Failed to load logs:', error);
    }
  };

  // Generate mock logs
  const mockLogs = Array.from({ length: 50 }, (_, i) => ({
    timestamp: new Date(Date.now() - i * 60000).toISOString(),
    level: ['INFO', 'WARN', 'ERROR', 'DEBUG'][Math.floor(Math.random() * 4)],
    node: `node-${Math.floor(Math.random() * 3) + 1}`,
    message: `PostgreSQL log message ${i}: Checkpoint completed successfully`,
  }));

  const displayLogs = logs.length > 0 ? logs : mockLogs;
  const filteredLogs = displayLogs.filter(log =>
    log.message.toLowerCase().includes(filter.toLowerCase()) ||
    log.node.toLowerCase().includes(filter.toLowerCase())
  );

  return (
    <div style={{ maxWidth: '1400px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <h2 style={{ margin: 0, fontSize: '20px' }}>Cluster Logs</h2>
        <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '14px' }}>
            <input type="checkbox" checked={autoRefresh} onChange={(e) => setAutoRefresh(e.target.checked)} />
            Auto-refresh (5s)
          </label>
          <button className="btn-secondary" onClick={loadLogs}>
            <RefreshCw size={16} />
            Refresh
          </button>
          <button className="btn-secondary">
            <Download size={16} />
            Download
          </button>
        </div>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <div style={{ position: 'relative' }}>
          <Search size={18} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
          <input
            type="text"
            placeholder="Search logs..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            style={{ width: '100%', padding: '10px 10px 10px 40px', border: '1px solid var(--border-color)', borderRadius: '8px', fontSize: '14px' }}
          />
        </div>
      </div>

      <div style={{ background: '#1e1e1e', borderRadius: '8px', padding: '16px', maxHeight: '600px', overflow: 'auto', fontFamily: 'JetBrains Mono, monospace', fontSize: '13px' }}>
        {filteredLogs.map((log, idx) => {
          const colors = { INFO: '#3b82f6', WARN: '#f59e0b', ERROR: '#ef4444', DEBUG: '#8b5cf6' };
          return (
            <div key={idx} style={{ marginBottom: '8px', display: 'flex', gap: '12px', color: '#d4d4d4' }}>
              <span style={{ color: '#6b7280', minWidth: '180px' }}>{new Date(log.timestamp).toLocaleString()}</span>
              <span style={{ color: colors[log.level] || '#3b82f6', fontWeight: 'bold', minWidth: '60px' }}>[{log.level}]</span>
              <span style={{ color: '#10b981', minWidth: '100px' }}>{log.node}</span>
              <span>{log.message}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default ClusterLogsTab;
