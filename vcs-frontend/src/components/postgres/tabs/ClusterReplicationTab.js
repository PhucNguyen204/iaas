import React, { useState, useEffect } from 'react';
import { GitBranch, RefreshCw, AlertTriangle, CheckCircle, Clock, Server } from 'lucide-react';
import { clusterAPI } from '../../../api';
import toast from 'react-hot-toast';
import './ClusterReplicationTab.css';

const ClusterReplicationTab = ({ clusterId, cluster }) => {
  const [replicationStatus, setReplicationStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    loadReplicationStatus();

    if (autoRefresh) {
      const interval = setInterval(loadReplicationStatus, 5000);
      return () => clearInterval(interval);
    }
  }, [clusterId, autoRefresh]);

  const loadReplicationStatus = async () => {
    try {
      setLoading(true);
      const response = await clusterAPI.getReplication(clusterId);
      setReplicationStatus(response.data.data);
    } catch (error) {
      console.error('Failed to load replication status:', error);
      toast.error('Failed to load replication status');
    } finally {
      setLoading(false);
    }
  };

  const formatLag = (lagBytes) => {
    if (lagBytes < 1024) return `${lagBytes} B`;
    if (lagBytes < 1048576) return `${(lagBytes / 1024).toFixed(1)} KB`;
    return `${(lagBytes / 1048576).toFixed(2)} MB`;
  };

  const getLagStatus = (lagBytes) => {
    if (lagBytes < 1048576) return 'healthy'; // < 1MB
    if (lagBytes < 10485760) return 'warning'; // < 10MB
    return 'critical'; // >= 10MB
  };

  if (loading && !replicationStatus) {
    return (
      <div className="replication-loading">
        <div className="spinner"></div>
        <p>Loading replication status...</p>
      </div>
    );
  }

  const primary = replicationStatus?.primary || cluster.nodes?.find(n => n.role === 'primary')?.node_name;
  const replicas = replicationStatus?.replicas || [];

  return (
    <div className="cluster-replication-tab">
      <div className="replication-header">
        <h2>Replication Monitor</h2>
        <div className="header-actions">
          <label className="auto-refresh-toggle">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            <span>Auto-refresh (5s)</span>
          </label>
          <button className="btn-secondary" onClick={loadReplicationStatus}>
            <RefreshCw size={16} />
            Refresh
          </button>
        </div>
      </div>

      <div className="replication-info">
        <div className="info-card">
          <div className="info-label">Replication Mode</div>
          <div className="info-value">
            <span className={`mode-badge ${cluster.replication_mode}`}>
              {cluster.replication_mode === 'sync' ? 'Synchronous' : 'Asynchronous'}
            </span>
          </div>
        </div>
        <div className="info-card">
          <div className="info-label">Primary Node</div>
          <div className="info-value">{primary || 'N/A'}</div>
        </div>
        <div className="info-card">
          <div className="info-label">Replica Nodes</div>
          <div className="info-value">{replicas.length}</div>
        </div>
        <div className="info-card">
          <div className="info-label">Max Allowed Lag</div>
          <div className="info-value">
            {cluster.max_replication_lag ? formatLag(cluster.max_replication_lag) : 'N/A'}
          </div>
        </div>
      </div>

      <div className="topology-diagram">
        <h3>Replication Topology</h3>
        
        <div className="topology-visual">
          {/* Primary Node */}
          <div className="primary-node-container">
            <div className="node-box primary-node">
              <div className="node-icon-wrapper">
                <Server size={32} />
              </div>
              <div className="node-info">
                <div className="node-label">PRIMARY (Master)</div>
                <div className="node-name">{primary}</div>
                <div className="node-badge primary">Read/Write</div>
              </div>
              <div className="node-status healthy">
                <CheckCircle size={20} />
              </div>
            </div>
          </div>

          {/* Replication Lines */}
          {replicas.length > 0 && (
            <div className="replication-lines">
              {replicas.map((replica, idx) => (
                <div key={idx} className={`replication-line ${getLagStatus(replica.lag_bytes)}`}>
                  <div className="line-arrow">→</div>
                  <div className="line-label">
                    {replica.sync_state === 'sync' ? 'Sync' : 'Async'}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Replica Nodes */}
          {replicas.length > 0 ? (
            <div className="replica-nodes-container">
              {replicas.map((replica, idx) => {
                const lagStatus = getLagStatus(replica.lag_bytes);
                return (
                  <div key={idx} className={`node-box replica-node ${lagStatus}`}>
                    <div className="node-icon-wrapper">
                      <Server size={28} />
                    </div>
                    <div className="node-info">
                      <div className="node-label">REPLICA (Standby {idx + 1})</div>
                      <div className="node-name">{replica.node_name}</div>
                      <div className="node-badge replica">Read-Only</div>
                    </div>
                    <div className={`node-status ${lagStatus}`}>
                      {replica.is_healthy ? (
                        <CheckCircle size={18} />
                      ) : (
                        <AlertTriangle size={18} />
                      )}
                    </div>
                    <div className="replication-details">
                      <div className="detail-row">
                        <Clock size={14} />
                        <span>Lag: {formatLag(replica.lag_bytes)}</span>
                      </div>
                      <div className="detail-row">
                        <GitBranch size={14} />
                        <span>State: {replica.state}</span>
                      </div>
                      <div className="detail-row">
                        <span className={`sync-badge ${replica.sync_state}`}>
                          {replica.sync_state === 'sync' ? 'Synchronous' : 'Asynchronous'}
                        </span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="no-replicas">
              <AlertTriangle size={32} />
              <p>No replica nodes configured</p>
            </div>
          )}
        </div>
      </div>

      {replicas.length > 0 && (
        <div className="replication-table">
          <h3>Detailed Replication Status</h3>
          <table>
            <thead>
              <tr>
                <th>Node Name</th>
                <th>State</th>
                <th>Sync Mode</th>
                <th>Lag (Bytes)</th>
                <th>Lag (Seconds)</th>
                <th>Health</th>
              </tr>
            </thead>
            <tbody>
              {replicas.map((replica, idx) => {
                const lagStatus = getLagStatus(replica.lag_bytes);
                return (
                  <tr key={idx} className={lagStatus}>
                    <td className="node-name-cell">
                      <Server size={16} />
                      {replica.node_name}
                    </td>
                    <td>
                      <span className="state-badge">{replica.state}</span>
                    </td>
                    <td>
                      <span className={`sync-badge ${replica.sync_state}`}>
                        {replica.sync_state}
                      </span>
                    </td>
                    <td className={`lag-cell ${lagStatus}`}>
                      {formatLag(replica.lag_bytes)}
                    </td>
                    <td>{replica.lag_seconds?.toFixed(2) || '0.00'}s</td>
                    <td className="health-cell">
                      {replica.is_healthy ? (
                        <span className="health-badge healthy">
                          <CheckCircle size={14} />
                          Healthy
                        </span>
                      ) : (
                        <span className="health-badge unhealthy">
                          <AlertTriangle size={14} />
                          Unhealthy
                        </span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className="legend">
        <h4>Status Legend</h4>
        <div className="legend-items">
          <div className="legend-item">
            <div className="legend-dot healthy"></div>
            <span>Healthy (&lt; 1 MB lag)</span>
          </div>
          <div className="legend-item">
            <div className="legend-dot warning"></div>
            <span>Warning (1-10 MB lag)</span>
          </div>
          <div className="legend-item">
            <div className="legend-dot critical"></div>
            <span>Critical (&gt; 10 MB lag)</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ClusterReplicationTab;
