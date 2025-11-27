import React, { useState, useEffect } from 'react';
import { Server, Activity, HardDrive, Cpu, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import { clusterAPI } from '../../../api';
import StatusBadge from '../../common/StatusBadge';
import './ClusterOverviewTab.css';

const ClusterOverviewTab = ({ cluster, onRefresh }) => {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
  }, [cluster.cluster_id]);

  const loadStats = async () => {
    try {
      setLoading(true);
      const response = await clusterAPI.getStats(cluster.cluster_id || cluster.id);
      setStats(response.data.data);
    } catch (error) {
      console.error('Failed to load stats:', error);
    } finally {
      setLoading(false);
    }
  };

  const getHealthStatus = () => {
    if (!cluster.nodes) return { percent: 0, status: 'Unknown' };
    const healthyNodes = cluster.nodes.filter(n => n.is_healthy).length;
    const percent = (healthyNodes / cluster.nodes.length) * 100;
    
    if (percent === 100) return { percent, status: 'Excellent' };
    if (percent >= 80) return { percent, status: 'Good' };
    if (percent >= 50) return { percent, status: 'Degraded' };
    return { percent, status: 'Critical' };
  };

  const health = getHealthStatus();
  const primaryNode = cluster.nodes?.find(n => n.role === 'primary');
  const replicaNodes = cluster.nodes?.filter(n => n.role === 'replica') || [];

  return (
    <div className="cluster-overview-tab">
      <div className="overview-header">
        <h2>Cluster Health Overview</h2>
        <button className="btn-secondary" onClick={() => { loadStats(); onRefresh(); }}>
          <Activity size={16} />
          Refresh
        </button>
      </div>

      <div className="health-summary">
        <div className="health-card main-health">
          <div className="health-icon">
            {health.percent >= 80 ? (
              <CheckCircle size={40} className="text-success" />
            ) : health.percent >= 50 ? (
              <AlertTriangle size={40} className="text-warning" />
            ) : (
              <XCircle size={40} className="text-error" />
            )}
          </div>
          <div className="health-info">
            <h3>Overall Health</h3>
            <div className="health-percent">{health.percent.toFixed(0)}%</div>
            <p className={`health-status status-${health.status.toLowerCase()}`}>{health.status}</p>
          </div>
        </div>

        <div className="health-card">
          <Server size={24} className="card-icon" />
          <div className="card-content">
            <div className="card-label">Active Nodes</div>
            <div className="card-value">{cluster.nodes?.filter(n => n.status === 'running').length || 0} / {cluster.nodes?.length || 0}</div>
          </div>
        </div>

        <div className="health-card">
          <HardDrive size={24} className="card-icon" />
          <div className="card-content">
            <div className="card-label">Total Databases</div>
            <div className="card-value">{stats?.total_databases || 0}</div>
          </div>
        </div>

        <div className="health-card">
          <Activity size={24} className="card-icon" />
          <div className="card-content">
            <div className="card-label">Total Connections</div>
            <div className="card-value">{stats?.total_connections || 0}</div>
          </div>
        </div>
      </div>

      <div className="nodes-section">
        <h3>Cluster Nodes</h3>

        {primaryNode && (
          <div className="node-group">
            <div className="node-group-header">
              <h4>Primary Node</h4>
              <span className="node-badge primary">MASTER</span>
            </div>
            <div className="node-card primary-node">
              <div className="node-header">
                <div className="node-name">
                  <Server size={20} />
                  <span>{primaryNode.node_name}</span>
                </div>
                <StatusBadge status={primaryNode.status} />
              </div>
              <div className="node-details">
                <div className="node-detail-item">
                  <span className="label">Container ID:</span>
                  <code>{primaryNode.container_id?.substring(0, 12)}</code>
                </div>
                <div className="node-detail-item">
                  <span className="label">Role:</span>
                  <span className="value primary-badge">Primary (Read/Write)</span>
                </div>
                <div className="node-detail-item">
                  <span className="label">Health:</span>
                  <span className={`value ${primaryNode.is_healthy ? 'text-success' : 'text-error'}`}>
                    {primaryNode.is_healthy ? '✓ Healthy' : '✗ Unhealthy'}
                  </span>
                </div>
                {stats?.nodes && stats.nodes.find(n => n.node_name === primaryNode.node_name) && (
                  <>
                    <div className="node-detail-item">
                      <span className="label">CPU Usage:</span>
                      <span className="value">{stats.nodes.find(n => n.node_name === primaryNode.node_name).cpu_percent}%</span>
                    </div>
                    <div className="node-detail-item">
                      <span className="label">Memory Usage:</span>
                      <span className="value">{stats.nodes.find(n => n.node_name === primaryNode.node_name).memory_percent}%</span>
                    </div>
                    <div className="node-detail-item">
                      <span className="label">Active Connections:</span>
                      <span className="value">{stats.nodes.find(n => n.node_name === primaryNode.node_name).active_connections}</span>
                    </div>
                  </>
                )}
              </div>
            </div>
          </div>
        )}

        {replicaNodes.length > 0 && (
          <div className="node-group">
            <div className="node-group-header">
              <h4>Replica Nodes</h4>
              <span className="node-badge replica">{replicaNodes.length} STANDBY</span>
            </div>
            <div className="replica-nodes-grid">
              {replicaNodes.map(node => {
                const nodeStats = stats?.nodes?.find(n => n.node_name === node.node_name);
                return (
                  <div key={node.node_id} className="node-card replica-node">
                    <div className="node-header">
                      <div className="node-name">
                        <Server size={18} />
                        <span>{node.node_name}</span>
                      </div>
                      <StatusBadge status={node.status} />
                    </div>
                    <div className="node-details">
                      <div className="node-detail-item">
                        <span className="label">Role:</span>
                        <span className="value replica-badge">Replica (Read-Only)</span>
                      </div>
                      <div className="node-detail-item">
                        <span className="label">Replication Lag:</span>
                        <span className={`value ${node.replication_delay > 1048576 ? 'text-warning' : 'text-success'}`}>
                          {(node.replication_delay / 1024).toFixed(0)} KB
                        </span>
                      </div>
                      <div className="node-detail-item">
                        <span className="label">Health:</span>
                        <span className={`value ${node.is_healthy ? 'text-success' : 'text-error'}`}>
                          {node.is_healthy ? '✓ Healthy' : '✗ Unhealthy'}
                        </span>
                      </div>
                      {nodeStats && (
                        <>
                          <div className="node-detail-item">
                            <span className="label">CPU:</span>
                            <span className="value">{nodeStats.cpu_percent}%</span>
                          </div>
                          <div className="node-detail-item">
                            <span className="label">Memory:</span>
                            <span className="value">{nodeStats.memory_percent}%</span>
                          </div>
                          <div className="node-detail-item">
                            <span className="label">Connections:</span>
                            <span className="value">{nodeStats.active_connections}</span>
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>

      <div className="connection-info">
        <h3>Connection Endpoints</h3>
        <div className="endpoints-grid">
          {cluster.write_endpoint && (
            <div className="endpoint-card primary-endpoint">
              <div className="endpoint-header">
                <span className="endpoint-label">Write Endpoint (Primary)</span>
                <span className="endpoint-badge primary">R/W</span>
              </div>
              <div className="endpoint-details">
                <div className="endpoint-url">
                  <code>postgresql://{cluster.write_endpoint.host}:{cluster.write_endpoint.port}</code>
                </div>
                <div className="endpoint-info">
                  <span>Node: {cluster.write_endpoint.node_id}</span>
                </div>
              </div>
            </div>
          )}

          {cluster.read_endpoints && cluster.read_endpoints.length > 0 && (
            <div className="endpoint-card replica-endpoint">
              <div className="endpoint-header">
                <span className="endpoint-label">Read Endpoints (Replicas)</span>
                <span className="endpoint-badge replica">R/O</span>
              </div>
              {cluster.read_endpoints.map((endpoint, idx) => (
                <div key={idx} className="endpoint-details">
                  <div className="endpoint-url">
                    <code>postgresql://{endpoint.host}:{endpoint.port}</code>
                  </div>
                  <div className="endpoint-info">
                    <span>Node: {endpoint.node_id}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ClusterOverviewTab;
