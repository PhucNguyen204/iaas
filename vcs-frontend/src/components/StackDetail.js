import React, { useState, useEffect, useCallback } from 'react';
import { 
  ArrowLeft, RefreshCw, Play, Square, RotateCcw, Trash2, 
  Database, Globe, Container, Server, ChevronDown, ChevronRight,
  FileText, Terminal, Copy, Download, AlertTriangle, CheckCircle,
  XCircle, Loader, Crown, Users, Plus, Minus
} from 'lucide-react';
import { stackAPI, clusterAPI, nginxAPI, dockerAPI } from '../api';
import toast from 'react-hot-toast';
import './StackDetail.css';

// Simple Status Badge
const StatusBadge = ({ status }) => {
  const getClass = () => {
    switch (status?.toLowerCase()) {
      case 'running': return 'badge badge-success';
      case 'stopped': return 'badge badge-warning';
      case 'failed': return 'badge badge-danger';
      case 'creating': return 'badge badge-info';
      default: return 'badge badge-default';
    }
  };
  return <span className={getClass()}>{status || 'unknown'}</span>;
};

// Role Badge for cluster nodes
const RoleBadge = ({ role }) => {
  if (role === 'leader' || role === 'primary') {
    return <span className="badge badge-primary"><Crown size={12} /> Primary</span>;
  }
  return <span className="badge badge-secondary"><Users size={12} /> Replica</span>;
};

const StackDetail = ({ stackId, onBack }) => {
  const [stack, setStack] = useState(null);
  const [loading, setLoading] = useState(true);
  const [expandedResources, setExpandedResources] = useState({});
  const [resourceDetails, setResourceDetails] = useState({});
  const [loadingDetails, setLoadingDetails] = useState({});
  const [actionLoading, setActionLoading] = useState({});
  const [activeTab, setActiveTab] = useState('overview'); // overview, logs
  const [logs, setLogs] = useState({});

  const loadStack = useCallback(async () => {
    try {
      setLoading(true);
      const response = await stackAPI.getById(stackId);
      setStack(response.data?.data || response.data);
    } catch (error) {
      console.error('Error loading stack:', error);
      toast.error('Failed to load stack');
    } finally {
      setLoading(false);
    }
  }, [stackId]);

  useEffect(() => {
    loadStack();
  }, [loadStack]);

  const toggleResource = async (resource) => {
    const key = resource.id || resource.resource_name;
    const isExpanded = expandedResources[key];
    
    setExpandedResources(prev => ({ ...prev, [key]: !isExpanded }));
    
    if (!isExpanded && !resourceDetails[key]) {
      await loadResourceDetails(resource);
    }
  };

  const loadResourceDetails = async (resource) => {
    const key = resource.id || resource.resource_name;
    const infraId = resource.outputs?.cluster_id || resource.infrastructure_id;
    
    if (!infraId) return;
    
    setLoadingDetails(prev => ({ ...prev, [key]: true }));
    try {
      let response;
      switch (resource.resource_type) {
        case 'POSTGRES_CLUSTER':
          response = await clusterAPI.getById(infraId);
          break;
        case 'NGINX_GATEWAY':
          response = await nginxAPI.getById(infraId);
          break;
        case 'DOCKER_SERVICE':
          response = await dockerAPI.getById(infraId);
          break;
        default:
          return;
      }
      setResourceDetails(prev => ({
        ...prev,
        [key]: response.data?.data || response.data
      }));
    } catch (error) {
      console.error('Error loading details:', error);
      toast.error('Failed to load resource details');
    } finally {
      setLoadingDetails(prev => ({ ...prev, [key]: false }));
    }
  };

  const loadLogs = async (resource) => {
    const key = resource.id || resource.resource_name;
    const infraId = resource.outputs?.cluster_id || resource.infrastructure_id;
    
    if (!infraId) return;
    
    try {
      let response;
      let logData;
      switch (resource.resource_type) {
        case 'POSTGRES_CLUSTER':
          response = await clusterAPI.getLogs(infraId, 100);
          // Backend returns: { data: { cluster_id, logs: [{ node_name, timestamp, logs }] } }
          logData = response.data?.data?.logs || response.data?.logs || [];
          break;
        case 'NGINX_GATEWAY':
          response = await nginxAPI.getLogs(infraId, 100);
          // Backend returns: { data: logs_string }
          logData = response.data?.data || response.data;
          break;
        case 'DOCKER_SERVICE':
          response = await dockerAPI.getLogs(infraId, 100);
          // Backend returns: { data: { logs: ["line1", "line2"] } }
          logData = response.data?.data?.logs || response.data?.logs || response.data?.data || response.data;
          break;
        default:
          return;
      }
      setLogs(prev => ({
        ...prev,
        [key]: logData
      }));
      toast.success('Logs loaded');
    } catch (error) {
      console.error('Error loading logs:', error);
      toast.error('Failed to load logs: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleAction = async (actionKey, action, resource, nodeId = null) => {
    setActionLoading(prev => ({ ...prev, [actionKey]: true }));
    const infraId = resource.outputs?.cluster_id || resource.infrastructure_id;
    
    try {
      switch (resource.resource_type) {
        case 'POSTGRES_CLUSTER':
          if (action === 'start') await clusterAPI.start(infraId);
          else if (action === 'stop') await clusterAPI.stop(infraId);
          else if (action === 'restart') await clusterAPI.restart(infraId);
          else if (action === 'failover') await clusterAPI.failover(infraId, { new_primary_node_id: nodeId });
          else if (action === 'backup') await clusterAPI.backup(infraId, { type: 'full' });
          else if (action === 'scale_up') await clusterAPI.scale(infraId, { node_count: (resourceDetails[resource.id]?.nodes?.length || 1) + 1 });
          else if (action === 'scale_down') await clusterAPI.scale(infraId, { node_count: Math.max(1, (resourceDetails[resource.id]?.nodes?.length || 2) - 1) });
          break;
        case 'NGINX_GATEWAY':
          if (action === 'start') await nginxAPI.start(infraId);
          else if (action === 'stop') await nginxAPI.stop(infraId);
          else if (action === 'restart') await nginxAPI.restart(infraId);
          else if (action === 'reload') await nginxAPI.reload(infraId);
          break;
        case 'DOCKER_SERVICE':
          if (action === 'start') await dockerAPI.start(infraId);
          else if (action === 'stop') await dockerAPI.stop(infraId);
          else if (action === 'restart') await dockerAPI.restart(infraId);
          break;
        default:
          break;
      }
      toast.success(`${action} successful`);
      await loadResourceDetails(resource);
    } catch (error) {
      toast.error(`${action} failed: ${error.message}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [actionKey]: false }));
    }
  };

  const handleStackAction = async (action) => {
    try {
      if (action === 'start') await stackAPI.start(stackId);
      else if (action === 'stop') await stackAPI.stop(stackId);
      else if (action === 'restart') await stackAPI.restart(stackId);
      else if (action === 'delete') {
        if (window.confirm('Delete this stack?')) {
          await stackAPI.delete(stackId);
          toast.success('Stack deleted');
          onBack();
          return;
        } else {
          return;
        }
      }
      toast.success(`Stack ${action} successful`);
      loadStack();
    } catch (error) {
      toast.error(`Stack ${action} failed`);
    }
  };

  if (loading) {
    return (
      <div className="stack-detail">
        <div className="loading-container">
          <Loader size={32} className="spin" />
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (!stack) {
    return (
      <div className="stack-detail">
        <div className="error-container">
          <AlertTriangle size={48} />
          <p>Stack not found</p>
          <button className="btn btn-primary" onClick={onBack}>Back</button>
        </div>
      </div>
    );
  }

  const resources = stack.resources || [];

  return (
    <div className="stack-detail">
      {/* Header */}
      <header className="detail-header">
        <button className="btn-back" onClick={onBack}>
          <ArrowLeft size={20} /> Back
        </button>
        <div className="header-info">
          <h1>{stack.name}</h1>
          <StatusBadge status={stack.status} />
          <span className="env-tag">{stack.environment}</span>
        </div>
        <div className="header-actions">
          <button className="btn btn-icon" onClick={loadStack} title="Refresh"><RefreshCw size={18} /></button>
          <button className="btn btn-success" onClick={() => handleStackAction('start')}><Play size={16} /> Start</button>
          <button className="btn btn-warning" onClick={() => handleStackAction('stop')}><Square size={16} /> Stop</button>
          <button className="btn btn-secondary" onClick={() => handleStackAction('restart')}><RotateCcw size={16} /> Restart</button>
          <button className="btn btn-danger" onClick={() => handleStackAction('delete')}><Trash2 size={16} /></button>
        </div>
      </header>

      {/* Tabs */}
      <div className="tabs">
        <button className={`tab ${activeTab === 'overview' ? 'active' : ''}`} onClick={() => setActiveTab('overview')}>
          Overview
        </button>
        <button className={`tab ${activeTab === 'logs' ? 'active' : ''}`} onClick={() => setActiveTab('logs')}>
          Logs
        </button>
      </div>

      {/* Content */}
      <div className="detail-content">
        {activeTab === 'overview' && (
          <div className="resources-list">
            <h2>Resources ({resources.length})</h2>
            {resources.length === 0 ? (
              <div className="empty">No resources</div>
            ) : (
              resources.map((resource, idx) => {
                const key = resource.id || `r-${idx}`;
                const isExpanded = expandedResources[key];
                const details = resourceDetails[key];
                const isLoading = loadingDetails[key];

                return (
                  <div key={key} className="resource-card">
                    {/* Resource Header */}
                    <div className="resource-header" onClick={() => toggleResource({ ...resource, id: key })}>
                      <div className="resource-title">
                        {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
                        <ResourceIcon type={resource.resource_type} />
                        <span className="name">{resource.resource_name}</span>
                        <span className="type">{resource.resource_type?.replace(/_/g, ' ')}</span>
                      </div>
                      <StatusBadge status={resource.status} />
                    </div>

                    {/* Expanded Details */}
                    {isExpanded && (
                      <div className="resource-body">
                        {isLoading ? (
                          <div className="loading"><Loader size={20} className="spin" /> Loading...</div>
                        ) : details ? (
                          <ResourceDetails 
                            type={resource.resource_type}
                            resource={resource}
                            details={details}
                            onAction={(action, nodeId) => handleAction(`${key}-${action}`, action, { ...resource, id: key }, nodeId)}
                            actionLoading={actionLoading}
                            actionKey={key}
                          />
                        ) : (
                          <div className="error">Could not load details</div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        )}

        {activeTab === 'logs' && (
          <div className="logs-section">
            <h2>Logs</h2>
            {resources.map((resource, idx) => {
              const key = resource.id || `r-${idx}`;
              const resourceLogs = logs[key];

              return (
                <div key={key} className="log-card">
                  <div className="log-header">
                    <ResourceIcon type={resource.resource_type} />
                    <span>{resource.resource_name}</span>
                    <button 
                      className="btn btn-sm btn-secondary"
                      onClick={() => loadLogs({ ...resource, id: key })}
                    >
                      <RefreshCw size={14} /> Load Logs
                    </button>
                  </div>
                  <div className="log-content">
                    {resourceLogs ? (
                      <LogsDisplay logs={resourceLogs} resourceType={resource.resource_type} />
                    ) : (
                      <p className="no-logs">Click "Load Logs" to fetch logs from Docker container</p>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

// Resource Icon
const ResourceIcon = ({ type }) => {
  switch (type?.toUpperCase()) {
    case 'POSTGRES_CLUSTER': return <Database size={18} className="icon-db" />;
    case 'NGINX_GATEWAY': return <Globe size={18} className="icon-nginx" />;
    case 'DOCKER_SERVICE': return <Container size={18} className="icon-docker" />;
    default: return <Server size={18} />;
  }
};

// Logs Display Component
const LogsDisplay = ({ logs, resourceType }) => {
  if (!logs) return null;
  
  // Handle cluster logs: array of { node_name, timestamp, logs }
  if (resourceType === 'POSTGRES_CLUSTER' && Array.isArray(logs)) {
    if (logs.length === 0) {
      return <p className="no-logs">No logs available</p>;
    }
    return (
      <div className="cluster-logs">
        {logs.map((nodeLog, idx) => (
          <div key={idx} className="node-log-section">
            <div className="node-log-header">
              <Database size={14} />
              <span className="node-name">{nodeLog.node_name}</span>
              {nodeLog.timestamp && <span className="timestamp">{nodeLog.timestamp}</span>}
            </div>
            <pre className="log-text">{nodeLog.logs || 'No logs'}</pre>
          </div>
        ))}
      </div>
    );
  }
  
  // Handle docker logs: array of strings
  if (Array.isArray(logs)) {
    if (logs.length === 0) {
      return <p className="no-logs">No logs available</p>;
    }
    return (
      <pre className="log-text">{logs.join('\n')}</pre>
    );
  }
  
  // Handle string logs (nginx)
  if (typeof logs === 'string') {
    return <pre className="log-text">{logs || 'No logs available'}</pre>;
  }
  
  // Fallback: JSON display
  return <pre className="log-text">{JSON.stringify(logs, null, 2)}</pre>;
};

// Resource Details Component
const ResourceDetails = ({ type, resource, details, onAction, actionLoading, actionKey }) => {
  switch (type) {
    case 'POSTGRES_CLUSTER':
      return <ClusterDetails details={details} onAction={onAction} actionLoading={actionLoading} actionKey={actionKey} />;
    case 'NGINX_GATEWAY':
      return <NginxDetails details={details} onAction={onAction} actionLoading={actionLoading} actionKey={actionKey} />;
    case 'DOCKER_SERVICE':
      return <DockerDetails details={details} onAction={onAction} actionLoading={actionLoading} actionKey={actionKey} />;
    default:
      return <div>Unknown resource type</div>;
  }
};

// PostgreSQL Cluster Details
const ClusterDetails = ({ details, onAction, actionLoading, actionKey }) => {
  const nodes = details.nodes || [];

  return (
    <div className="cluster-details">
      {/* Cluster Actions */}
      <div className="action-bar">
        <button 
          className="btn btn-success btn-sm" 
          onClick={() => onAction('start')}
          disabled={actionLoading[`${actionKey}-start`]}
        >
          {actionLoading[`${actionKey}-start`] ? <Loader size={14} className="spin" /> : <Play size={14} />} Start
        </button>
        <button 
          className="btn btn-warning btn-sm" 
          onClick={() => onAction('stop')}
          disabled={actionLoading[`${actionKey}-stop`]}
        >
          {actionLoading[`${actionKey}-stop`] ? <Loader size={14} className="spin" /> : <Square size={14} />} Stop
        </button>
        <button 
          className="btn btn-secondary btn-sm" 
          onClick={() => onAction('restart')}
          disabled={actionLoading[`${actionKey}-restart`]}
        >
          {actionLoading[`${actionKey}-restart`] ? <Loader size={14} className="spin" /> : <RotateCcw size={14} />} Restart
        </button>
        <button 
          className="btn btn-primary btn-sm" 
          onClick={() => onAction('backup')}
          disabled={actionLoading[`${actionKey}-backup`]}
        >
          {actionLoading[`${actionKey}-backup`] ? <Loader size={14} className="spin" /> : <Download size={14} />} Backup
        </button>
        <button 
          className="btn btn-sm" 
          onClick={() => onAction('scale_up')}
          disabled={actionLoading[`${actionKey}-scale_up`]}
        >
          <Plus size={14} /> Add Node
        </button>
        <button 
          className="btn btn-sm" 
          onClick={() => onAction('scale_down')}
          disabled={actionLoading[`${actionKey}-scale_down`] || nodes.length <= 1}
        >
          <Minus size={14} /> Remove Node
        </button>
      </div>

      {/* Cluster Info */}
      <div className="info-grid">
        <div className="info-item">
          <span className="label">Cluster Name</span>
          <span className="value">{details.cluster_name}</span>
        </div>
        <div className="info-item">
          <span className="label">Version</span>
          <span className="value">PostgreSQL {details.postgres_version}</span>
        </div>
        <div className="info-item">
          <span className="label">Replication</span>
          <span className="value">{details.replication_mode || 'async'}</span>
        </div>
        <div className="info-item">
          <span className="label">HAProxy Port</span>
          <span className="value">{details.haproxy_port || 'N/A'}</span>
        </div>
      </div>

      {/* Connection Endpoints */}
      {details.write_endpoint && (
        <div className="endpoints">
          <h4>Connection Endpoints</h4>
          <div className="endpoint">
            <span className="label">Write (Primary):</span>
            <code>{details.write_endpoint.host}:{details.write_endpoint.port}</code>
            <button className="btn-copy" onClick={() => {
              navigator.clipboard.writeText(`${details.write_endpoint.host}:${details.write_endpoint.port}`);
              toast.success('Copied!');
            }}>
              <Copy size={12} />
            </button>
          </div>
          {details.read_endpoints?.length > 0 && (
            <div className="endpoint">
              <span className="label">Read (Replicas):</span>
              <span>{details.read_endpoints.length} endpoints</span>
            </div>
          )}
        </div>
      )}

      {/* Nodes Table */}
      <div className="nodes-section">
        <h4>Cluster Nodes ({nodes.length})</h4>
        <table className="nodes-table">
          <thead>
            <tr>
              <th>Node Name</th>
              <th>Role</th>
              <th>Status</th>
              <th>Health</th>
              <th>Container ID</th>
              <th>Replication Lag</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {nodes.map((node, idx) => (
              <tr key={idx} className={node.role === 'leader' || node.role === 'primary' ? 'row-primary' : ''}>
                <td>
                  <div className="node-name">
                    <Database size={14} />
                    {node.node_name}
                  </div>
                </td>
                <td><RoleBadge role={node.role} /></td>
                <td><StatusBadge status={node.status} /></td>
                <td>
                  {node.is_healthy ? (
                    <CheckCircle size={16} className="icon-success" />
                  ) : (
                    <XCircle size={16} className="icon-error" />
                  )}
                </td>
                <td><code className="container-id">{node.container_id?.substring(0, 12)}</code></td>
                <td>{node.replication_delay ? `${node.replication_delay} bytes` : '-'}</td>
                <td>
                  <div className="node-actions">
                    {(node.role === 'replica' || node.role === 'follower') && (
                      <button 
                        className="btn btn-xs btn-danger"
                        onClick={() => {
                          if (window.confirm(`Promote ${node.node_name} to Primary? This will trigger failover.`)) {
                            onAction('failover', node.node_id);
                          }
                        }}
                        disabled={actionLoading[`${actionKey}-failover`]}
                        title="Promote to Primary (Failover)"
                      >
                        <Crown size={12} /> Failover
                      </button>
                    )}
                    <button className="btn btn-xs" title="View Logs">
                      <FileText size={12} />
                    </button>
                    <button className="btn btn-xs" title="Terminal">
                      <Terminal size={12} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

// Nginx Details
const NginxDetails = ({ details, onAction, actionLoading, actionKey }) => {
  return (
    <div className="nginx-details">
      {/* Actions */}
      <div className="action-bar">
        <button 
          className="btn btn-success btn-sm" 
          onClick={() => onAction('start')}
          disabled={actionLoading[`${actionKey}-start`]}
        >
          <Play size={14} /> Start
        </button>
        <button 
          className="btn btn-warning btn-sm" 
          onClick={() => onAction('stop')}
          disabled={actionLoading[`${actionKey}-stop`]}
        >
          <Square size={14} /> Stop
        </button>
        <button 
          className="btn btn-secondary btn-sm" 
          onClick={() => onAction('restart')}
          disabled={actionLoading[`${actionKey}-restart`]}
        >
          <RotateCcw size={14} /> Restart
        </button>
        <button 
          className="btn btn-primary btn-sm" 
          onClick={() => onAction('reload')}
          disabled={actionLoading[`${actionKey}-reload`]}
        >
          <RefreshCw size={14} /> Reload Config
        </button>
      </div>

      {/* Info */}
      <div className="info-grid">
        <div className="info-item">
          <span className="label">Name</span>
          <span className="value">{details.name}</span>
        </div>
        <div className="info-item">
          <span className="label">Port</span>
          <span className="value">{details.port}</span>
        </div>
        <div className="info-item">
          <span className="label">Container ID</span>
          <code className="value">{details.container_id?.substring(0, 12)}</code>
        </div>
        <div className="info-item">
          <span className="label">Status</span>
          <StatusBadge status={details.status} />
        </div>
      </div>

      {/* Domains */}
      {details.domains?.length > 0 && (
        <div className="domains">
          <h4>Domains</h4>
          <ul>
            {details.domains.map((d, i) => (
              <li key={i}>{d.domain_name}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

// Docker Service Details
const DockerDetails = ({ details, onAction, actionLoading, actionKey }) => {
  return (
    <div className="docker-details">
      {/* Actions */}
      <div className="action-bar">
        <button 
          className="btn btn-success btn-sm" 
          onClick={() => onAction('start')}
          disabled={actionLoading[`${actionKey}-start`]}
        >
          <Play size={14} /> Start
        </button>
        <button 
          className="btn btn-warning btn-sm" 
          onClick={() => onAction('stop')}
          disabled={actionLoading[`${actionKey}-stop`]}
        >
          <Square size={14} /> Stop
        </button>
        <button 
          className="btn btn-secondary btn-sm" 
          onClick={() => onAction('restart')}
          disabled={actionLoading[`${actionKey}-restart`]}
        >
          <RotateCcw size={14} /> Restart
        </button>
      </div>

      {/* Info */}
      <div className="info-grid">
        <div className="info-item">
          <span className="label">Name</span>
          <span className="value">{details.name}</span>
        </div>
        <div className="info-item">
          <span className="label">Image</span>
          <span className="value">{details.image}:{details.image_tag || 'latest'}</span>
        </div>
        <div className="info-item">
          <span className="label">Container ID</span>
          <code className="value">{details.container_id?.substring(0, 12)}</code>
        </div>
        <div className="info-item">
          <span className="label">Status</span>
          <StatusBadge status={details.status} />
        </div>
      </div>

      {/* Ports */}
      {details.ports?.length > 0 && (
        <div className="ports">
          <h4>Ports</h4>
          <ul>
            {details.ports.map((p, i) => (
              <li key={i}>{p.host_port}:{p.container_port} ({p.protocol || 'tcp'})</li>
            ))}
          </ul>
        </div>
      )}

      {/* Env Vars */}
      {details.env_vars?.length > 0 && (
        <div className="env-vars">
          <h4>Environment Variables</h4>
          <ul>
            {details.env_vars.map((e, i) => (
              <li key={i}><strong>{e.key}:</strong> {e.is_secret ? '••••••' : e.value}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default StackDetail;
