import React, { useState, useEffect } from 'react';
import { Server, RefreshCw, Power, Activity, AlertCircle, CheckCircle, Clock, HardDrive, Cpu, Database, Copy, ExternalLink } from 'lucide-react';
import { clusterAPI } from '../../../api';
import toast from 'react-hot-toast';
import StatusBadge from '../../common/StatusBadge';
import './ClusterNodesTab.css';

const ClusterNodesTab = ({ clusterId, cluster, onRefresh }) => {
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedNode, setSelectedNode] = useState(null);
  const [nodeStats, setNodeStats] = useState({});
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    loadNodes();
    
    if (autoRefresh) {
      const interval = setInterval(() => {
        loadNodes();
        loadNodeStats();
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [clusterId, autoRefresh]);

  useEffect(() => {
    if (cluster?.nodes) {
      setNodes(cluster.nodes);
      loadNodeStats();
    }
  }, [cluster]);

  const loadNodes = async () => {
    try {
      setLoading(true);
      const response = await clusterAPI.getById(clusterId);
      if (response.data.data?.nodes) {
        setNodes(response.data.data.nodes);
      }
    } catch (error) {
      console.error('Failed to load nodes:', error);
      toast.error('Failed to load cluster nodes');
    } finally {
      setLoading(false);
    }
  };

  const loadNodeStats = async () => {
    try {
      const response = await clusterAPI.getStats(clusterId);
      if (response.data.data?.nodes) {
        const statsMap = {};
        response.data.data.nodes.forEach(node => {
          statsMap[node.node_name] = node;
        });
        setNodeStats(statsMap);
      }
    } catch (error) {
      console.error('Failed to load node stats:', error);
    }
  };

  const formatBytes = (bytes) => {
    if (!bytes) return '0 B';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(2)} MB`;
    return `${(bytes / 1073741824).toFixed(2)} GB`;
  };

  const formatReplicationDelay = (delay) => {
    if (!delay) return '0 B';
    if (delay < 1024) return `${delay} B`;
    if (delay < 1048576) return `${(delay / 1024).toFixed(1)} KB`;
    return `${(delay / 1048576).toFixed(2)} MB`;
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };

  const getNodeConnectionString = (node) => {
    const endpoint = node.role === 'primary' 
      ? cluster.write_endpoint 
      : cluster.read_endpoints?.find(e => e.node_id === node.node_id);
    
    if (!endpoint) return null;
    
    const username = cluster.username || 'postgres';
    const password = cluster.password ? `:${cluster.password}` : '';
    const database = cluster.database_name || cluster.databaseName || 'postgres';
    
    return `postgresql://${username}${password}@${endpoint.host}:${endpoint.port}/${database}`;
  };

  const primaryNode = nodes.find(n => n.role === 'primary');
  const replicaNodes = nodes.filter(n => n.role === 'replica');

  if (loading && nodes.length === 0) {
    return (
      <div className="cluster-nodes-loading">
        <div className="spinner"></div>
        <p>Loading cluster nodes...</p>
      </div>
    );
  }

  return (
    <div className="cluster-nodes-tab">
      <div className="nodes-header">
        <div>
          <h2>Cluster Nodes Management</h2>
          <p className="nodes-subtitle">
            Manage and monitor individual nodes in your PostgreSQL cluster
          </p>
        </div>
        <div className="header-actions">
          <label className="auto-refresh-toggle">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
            />
            <span>Auto-refresh (5s)</span>
          </label>
          <button className="btn-secondary" onClick={() => { loadNodes(); loadNodeStats(); onRefresh(); }}>
            <RefreshCw size={16} />
            Refresh
          </button>
        </div>
      </div>

      <div className="nodes-summary">
        <div className="summary-card">
          <Server size={24} />
          <div>
            <div className="summary-label">Total Nodes</div>
            <div className="summary-value">{nodes.length}</div>
          </div>
        </div>
        <div className="summary-card">
          <CheckCircle size={24} className="text-success" />
          <div>
            <div className="summary-label">Healthy Nodes</div>
            <div className="summary-value">{nodes.filter(n => n.is_healthy).length}</div>
          </div>
        </div>
        <div className="summary-card">
          <Activity size={24} />
          <div>
            <div className="summary-label">Running Nodes</div>
            <div className="summary-value">{nodes.filter(n => n.status === 'running').length}</div>
          </div>
        </div>
        <div className="summary-card">
          <Database size={24} />
          <div>
            <div className="summary-label">Primary Node</div>
            <div className="summary-value">{primaryNode ? 'Active' : 'N/A'}</div>
          </div>
        </div>
      </div>

      {/* Primary Node Section */}
      {primaryNode && (
        <div className="node-section primary-section">
          <div className="section-header">
            <h3>
              <Server size={20} />
              Primary Node (Master)
            </h3>
            <span className="role-badge primary">Read/Write</span>
          </div>
          
          <div className="node-detail-card primary-node-card">
            <div className="node-card-header">
              <div className="node-identity">
                <div className="node-name-large">
                  <Server size={24} />
                  <div>
                    <h4>{primaryNode.node_name}</h4>
                    <code className="node-id">{primaryNode.node_id}</code>
                  </div>
                </div>
                <StatusBadge status={primaryNode.status} />
              </div>
              
              <div className="node-health-indicator">
                {primaryNode.is_healthy ? (
                  <div className="health-badge healthy">
                    <CheckCircle size={18} />
                    Healthy
                  </div>
                ) : (
                  <div className="health-badge unhealthy">
                    <AlertCircle size={18} />
                    Unhealthy
                  </div>
                )}
              </div>
            </div>

            <div className="node-details-grid">
              <div className="detail-group">
                <h5>Container Information</h5>
                <div className="detail-item">
                  <span className="detail-label">Container ID:</span>
                  <div className="detail-value-group">
                    <code>{primaryNode.container_id || 'N/A'}</code>
                    {primaryNode.container_id && (
                      <button 
                        className="btn-icon-sm" 
                        onClick={() => copyToClipboard(primaryNode.container_id)}
                        title="Copy"
                      >
                        <Copy size={14} />
                      </button>
                    )}
                  </div>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Role:</span>
                  <span className="detail-value primary-badge">Primary (Master)</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Status:</span>
                  <StatusBadge status={primaryNode.status} />
                </div>
              </div>

              <div className="detail-group">
                <h5>Performance Metrics</h5>
                {nodeStats[primaryNode.node_name] ? (
                  <>
                    <div className="detail-item">
                      <span className="detail-label">
                        <Cpu size={14} />
                        CPU Usage:
                      </span>
                      <div className="metric-value">
                        <span className="value">{nodeStats[primaryNode.node_name].cpu_percent}%</span>
                        <div className="progress-bar">
                          <div 
                            className="progress-fill" 
                            style={{ width: `${nodeStats[primaryNode.node_name].cpu_percent}%` }}
                          />
                        </div>
                      </div>
                    </div>
                    <div className="detail-item">
                      <span className="detail-label">
                        <HardDrive size={14} />
                        Memory Usage:
                      </span>
                      <div className="metric-value">
                        <span className="value">{nodeStats[primaryNode.node_name].memory_percent}%</span>
                        <div className="progress-bar">
                          <div 
                            className="progress-fill" 
                            style={{ width: `${nodeStats[primaryNode.node_name].memory_percent}%` }}
                          />
                        </div>
                      </div>
                    </div>
                    <div className="detail-item">
                      <span className="detail-label">
                        <Database size={14} />
                        Active Connections:
                      </span>
                      <span className="detail-value">{nodeStats[primaryNode.node_name].active_connections}</span>
                    </div>
                  </>
                ) : (
                  <div className="no-stats">No metrics available</div>
                )}
              </div>

              <div className="detail-group">
                <h5>Connection Endpoint</h5>
                {cluster.write_endpoint && (
                  <>
                    <div className="detail-item">
                      <span className="detail-label">Host:</span>
                      <div className="detail-value-group">
                        <code>{cluster.write_endpoint.host}</code>
                        <button 
                          className="btn-icon-sm" 
                          onClick={() => copyToClipboard(cluster.write_endpoint.host)}
                          title="Copy"
                        >
                          <Copy size={14} />
                        </button>
                      </div>
                    </div>
                    <div className="detail-item">
                      <span className="detail-label">Port:</span>
                      <code>{cluster.write_endpoint.port}</code>
                    </div>
                    {cluster.write_endpoint && (
                      <div className="detail-item">
                        <span className="detail-label">Connection String:</span>
                        <div className="connection-string">
                          <code>
                            {getNodeConnectionString(primaryNode) || 
                             `postgresql://${cluster.write_endpoint.host}:${cluster.write_endpoint.port}/postgres`}
                          </code>
                          <button 
                            className="btn-icon-sm" 
                            onClick={() => copyToClipboard(
                              getNodeConnectionString(primaryNode) || 
                              `postgresql://${cluster.write_endpoint.host}:${cluster.write_endpoint.port}/postgres`
                            )}
                            title="Copy"
                          >
                            <Copy size={14} />
                          </button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Replica Nodes Section */}
      {replicaNodes.length > 0 && (
        <div className="node-section replicas-section">
          <div className="section-header">
            <h3>
              <Server size={20} />
              Replica Nodes (Standby)
            </h3>
            <span className="role-badge replica">{replicaNodes.length} Replicas</span>
          </div>

          <div className="replica-nodes-grid">
            {replicaNodes.map((node) => {
              const stats = nodeStats[node.node_name];
              const endpoint = cluster.read_endpoints?.find(e => e.node_id === node.node_id);
              
              return (
                <div 
                  key={node.node_id} 
                  className={`node-detail-card replica-node-card ${selectedNode?.node_id === node.node_id ? 'selected' : ''}`}
                  onClick={() => setSelectedNode(selectedNode?.node_id === node.node_id ? null : node)}
                >
                  <div className="node-card-header">
                    <div className="node-identity">
                      <div className="node-name-large">
                        <Server size={20} />
                        <div>
                          <h4>{node.node_name}</h4>
                          <code className="node-id">{node.node_id.substring(0, 8)}...</code>
                        </div>
                      </div>
                      <StatusBadge status={node.status} />
                    </div>
                    
                    <div className="node-health-indicator">
                      {node.is_healthy ? (
                        <div className="health-badge healthy">
                          <CheckCircle size={16} />
                        </div>
                      ) : (
                        <div className="health-badge unhealthy">
                          <AlertCircle size={16} />
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="node-details-compact">
                    <div className="detail-row">
                      <span className="detail-label">Role:</span>
                      <span className="detail-value replica-badge">Replica (Read-Only)</span>
                    </div>
                    
                    <div className="detail-row">
                      <span className="detail-label">
                        <Clock size={12} />
                        Replication Lag:
                      </span>
                      <span className={`detail-value ${node.replication_delay > 1048576 ? 'text-warning' : 'text-success'}`}>
                        {formatReplicationDelay(node.replication_delay)}
                      </span>
                    </div>

                    {stats && (
                      <>
                        <div className="detail-row">
                          <span className="detail-label">
                            <Cpu size={12} />
                            CPU:
                          </span>
                          <span className="detail-value">{stats.cpu_percent}%</span>
                        </div>
                        <div className="detail-row">
                          <span className="detail-label">
                            <HardDrive size={12} />
                            Memory:
                          </span>
                          <span className="detail-value">{stats.memory_percent}%</span>
                        </div>
                        <div className="detail-row">
                          <span className="detail-label">
                            <Database size={12} />
                            Connections:
                          </span>
                          <span className="detail-value">{stats.active_connections}</span>
                        </div>
                      </>
                    )}

                    {endpoint && (
                      <div className="detail-row">
                        <span className="detail-label">Endpoint:</span>
                        <code className="endpoint-code">{endpoint.host}:{endpoint.port}</code>
                      </div>
                    )}
                  </div>

                  {selectedNode?.node_id === node.node_id && (
                    <div className="node-expanded-details">
                      <div className="expanded-section">
                        <h5>Container Information</h5>
                        <div className="detail-item">
                          <span className="detail-label">Container ID:</span>
                          <div className="detail-value-group">
                            <code>{node.container_id || 'N/A'}</code>
                            {node.container_id && (
                              <button 
                                className="btn-icon-sm" 
                                onClick={(e) => {
                                  e.stopPropagation();
                                  copyToClipboard(node.container_id);
                                }}
                                title="Copy"
                              >
                                <Copy size={14} />
                              </button>
                            )}
                          </div>
                        </div>
                      </div>

                      {endpoint && (
                        <div className="expanded-section">
                          <h5>Connection Details</h5>
                          <div className="detail-item">
                            <span className="detail-label">Host:</span>
                            <code>{endpoint.host}</code>
                          </div>
                          <div className="detail-item">
                            <span className="detail-label">Port:</span>
                            <code>{endpoint.port}</code>
                          </div>
                          {getNodeConnectionString(node) && (
                            <div className="detail-item">
                              <span className="detail-label">Connection String:</span>
                              <div className="connection-string">
                                <code>{getNodeConnectionString(node)}</code>
                                <button 
                                  className="btn-icon-sm" 
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    copyToClipboard(getNodeConnectionString(node));
                                  }}
                                  title="Copy"
                                >
                                  <Copy size={14} />
                                </button>
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {nodes.length === 0 && (
        <div className="no-nodes">
          <Server size={48} />
          <h3>No Nodes Found</h3>
          <p>This cluster doesn't have any nodes configured yet.</p>
        </div>
      )}
    </div>
  );
};

export default ClusterNodesTab;

