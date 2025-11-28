import React, { useState, useEffect } from 'react';
import { X, Database, RefreshCw, Play, Square, RotateCcw, Crown, Plus, Minus, Copy, Download, Globe, AlertTriangle, Activity, Edit3, Settings, Upload } from 'lucide-react';
import { postgresAPI, nginxAPI, clusterAPI, nginxClusterAPI, dinDAPI } from '../api';
import StatusBadge from './common/StatusBadge';
import ResourceIcon from './common/ResourceIcon';
import toast from 'react-hot-toast';
import './InfrastructureDetailModal.css';

const InfrastructureDetailModal = ({ isOpen, onClose, resource, onRefresh }) => {
  const [details, setDetails] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen && resource) {
      loadInfrastructureDetails();
    }
  }, [isOpen, resource]);

  const loadInfrastructureDetails = async () => {
    if (!resource || !resource.infrastructure_id) return;

    try {
      setLoading(true);
      let response;

      switch (resource.resource_type) {
        case 'NGINX_GATEWAY':
          response = await nginxAPI.getById(resource.infrastructure_id);
          setDetails(response.data?.data || response.data);
          break;
        case 'NGINX_CLUSTER': {
          const clusterId = resource.outputs?.cluster_id || resource.infrastructure_id;
          response = await nginxClusterAPI.getById(clusterId);
          setDetails(response.data?.data || response.data);
          break;
        }
        
        case 'POSTGRES_INSTANCE':
          response = await postgresAPI.getById(resource.infrastructure_id);
          setDetails(response.data?.data || response.data);
          break;
        
        case 'POSTGRES_CLUSTER':
          // For cluster, we need cluster_id from outputs, fallback to infrastructure_id
          const clusterId = resource.outputs?.cluster_id || resource.infrastructure_id;
          response = await clusterAPI.getById(clusterId);
          setDetails(response.data?.data || response.data);
          break;
        
        case 'DIND_ENVIRONMENT': {
          // For DinD, we need environment_id from outputs, fallback to infrastructure_id
          const envId = resource.outputs?.environment_id || resource.infrastructure_id;
          response = await dinDAPI.getEnvironment(envId);
          setDetails(response.data?.data || response.data);
          break;
        }
        
        default:
          toast.error('Unsupported infrastructure type');
          return;
      }
    } catch (error) {
      console.error('Error loading infrastructure details:', error);
      toast.error('Failed to load infrastructure details');
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen || !resource) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="infrastructure-detail-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-title">
            <ResourceIcon type={resource.resource_type} size={24} />
            <h2>{resource.resource_name || resource.resource_type}</h2>
            <StatusBadge status={resource.status} />
          </div>
          <button className="close-btn" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <div className="modal-content">
          {loading ? (
            <div className="loading-state">Loading details...</div>
          ) : details ? (
            <InfrastructureDetails 
              resourceType={resource.resource_type} 
              details={details} 
              onRefresh={() => {
                loadInfrastructureDetails();
                if (onRefresh) onRefresh();
              }}
              onClose={onClose}
            />
          ) : (
            <div className="error-state">Failed to load details</div>
          )}
        </div>
      </div>
    </div>
  );
};

const InfrastructureDetails = ({ resourceType, details, onRefresh, onClose }) => {
  switch (resourceType) {
    case 'NGINX_GATEWAY':
      return <NginxDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    case 'NGINX_CLUSTER':
      return <NginxClusterDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    case 'POSTGRES_INSTANCE':
      return <PostgresDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    case 'POSTGRES_CLUSTER':
      return <PostgresClusterDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    case 'DIND_ENVIRONMENT':
      return <DinDEnvironmentDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    default:
      return <div>Unsupported infrastructure type</div>;
  }
};

const NginxDetails = ({ details, onRefresh, onClose }) => {
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    if (window.confirm(`Delete NGINX "${details.name}"? This cannot be undone!`)) {
      setDeleting(true);
      try {
        await nginxAPI.delete(details.id);
        toast.success('NGINX deleted successfully');
        onClose();
        if (onRefresh) onRefresh();
      } catch (error) {
        toast.error(`Delete failed: ${error.response?.data?.message || error.message}`);
        setDeleting(false);
      }
    }
  };

  return (
    <div className="infrastructure-details">
      <div className="detail-section">
        <h3>Actions</h3>
        <div className="action-buttons">
          <button 
            className="btn btn-sm btn-danger" 
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? (
              <><RefreshCw size={14} className="spin" /> Deleting...</>
            ) : (
              <><X size={14} /> Delete</>
            )}
          </button>
        </div>
      </div>
      <div className="detail-section">
        <h3>Basic Information</h3>
        <div className="detail-grid">
          <div className="detail-item">
            <span className="label">Name:</span>
            <span className="value">{details.name}</span>
          </div>
          <div className="detail-item">
            <span className="label">Status:</span>
            <StatusBadge status={details.status} />
          </div>
          <div className="detail-item">
            <span className="label">Port:</span>
            <span className="value">{details.port}</span>
          </div>
          <div className="detail-item">
            <span className="label">Container ID:</span>
            <span className="value code">{details.container_id?.substring(0, 12)}</span>
          </div>
        </div>
      </div>

      {details.domains && details.domains.length > 0 && (
        <div className="detail-section">
          <h3>Domains</h3>
          <ul>
            {details.domains.map((domain, idx) => (
              <li key={idx}>{domain.domain_name}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

const NginxClusterDetails = ({ details, onRefresh, onClose }) => {
  const clusterId = details?.id || details?.cluster_id;
  const [clusterInfo, setClusterInfo] = useState(details);
  const [loading, setLoading] = useState(false);
  const [testResult, setTestResult] = useState(null);
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [configText, setConfigText] = useState(details?.nginx_config || '');
  const [savingConfig, setSavingConfig] = useState(false);
  const [syncingConfig, setSyncingConfig] = useState(false);

  const reload = async () => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const resp = await nginxClusterAPI.getById(clusterId);
      const data = resp.data?.data || resp.data;
      setClusterInfo(data);
      setTestResult(null);
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(error.response?.data?.message || 'Failed to refresh cluster info');
    } finally {
      setLoading(false);
    }
  };

  const handleAction = async (action) => {
    if (!clusterId) return;
    setLoading(true);
    try {
      switch (action) {
        case 'start':
          await nginxClusterAPI.start(clusterId);
          toast.success('Cluster started');
          break;
        case 'stop':
          await nginxClusterAPI.stop(clusterId);
          toast.success('Cluster stopped');
          break;
        case 'restart':
          await nginxClusterAPI.restart(clusterId);
          toast.success('Cluster restarted');
          break;
        case 'test':
          const resp = await nginxClusterAPI.testConnection(clusterId);
          const result = resp.data?.data || resp.data;
          setTestResult(result);
          if (result?.success) {
            toast.success(`Connection OK via ${result?.node_name || 'master'}`);
          } else {
            toast.error(result?.message || 'Test failed');
          }
          setLoading(false);
          return;
        default:
          break;
      }
      await reload();
    } catch (error) {
      toast.error(error.response?.data?.message || `Failed to ${action} cluster`);
      setLoading(false);
    }
  };

  const handleOpenConfigModal = () => {
    setConfigText(clusterInfo?.nginx_config || getDefaultNginxConfig());
    setShowConfigModal(true);
  };

  const handleSaveConfig = async (reloadAll = false) => {
    if (!clusterId || !configText.trim()) return;
    setSavingConfig(true);
    try {
      await nginxClusterAPI.updateConfig(clusterId, {
        nginx_config: configText,
        reload_all: reloadAll
      });
      toast.success(reloadAll ? 'Config saved and synced to all nodes!' : 'Config saved successfully!');
      setShowConfigModal(false);
      await reload();
    } catch (error) {
      toast.error(error.response?.data?.message || 'Failed to save config');
    } finally {
      setSavingConfig(false);
    }
  };

  const handleSyncConfig = async () => {
    if (!clusterId) return;
    setSyncingConfig(true);
    try {
      await nginxClusterAPI.syncConfig(clusterId);
      toast.success('Config synced to all nodes!');
      await reload();
    } catch (error) {
      toast.error(error.response?.data?.message || 'Failed to sync config');
    } finally {
      setSyncingConfig(false);
    }
  };

  const getDefaultNginxConfig = () => {
    return `# Nginx Configuration for ${clusterInfo?.cluster_name || 'cluster'}
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections ${clusterInfo?.worker_connections || 2048};
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout ${clusterInfo?.keepalive_timeout || 65};
    types_hash_max_size 2048;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/xml;

    # Default server
    server {
        listen ${clusterInfo?.http_port || 80};
        server_name localhost;

        location / {
            root /usr/share/nginx/html;
            index index.html index.htm;
        }

        location /health {
            access_log off;
            return 200 "healthy\\n";
            add_header Content-Type text/plain;
        }
    }
}`;
  };

  if (!clusterInfo) {
    return (
      <div className="infrastructure-details">
        <div className="detail-section">
          <div className="detail-grid">
            <div className="detail-item">
              <span className="label">Cluster</span>
              <span className="value">Not available</span>
            </div>
            <button className="btn btn-secondary btn-sm" onClick={reload}>
              <RefreshCw size={14} /> Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  const nodes = clusterInfo.nodes || [];

  return (
    <div className="infrastructure-details">
      <div className="detail-section">
        <h3>Cluster Actions</h3>
        <div className="action-buttons">
          <button className="btn btn-sm btn-secondary" onClick={() => handleAction('start')} disabled={loading}>
            {loading ? <RefreshCw size={14} className="spin" /> : <Play size={14} />} Start
          </button>
          <button className="btn btn-sm btn-secondary" onClick={() => handleAction('stop')} disabled={loading}>
            {loading ? <RefreshCw size={14} className="spin" /> : <Square size={14} />} Stop
          </button>
          <button className="btn btn-sm btn-secondary" onClick={() => handleAction('restart')} disabled={loading}>
            {loading ? <RefreshCw size={14} className="spin" /> : <RotateCcw size={14} />} Restart
          </button>
          <button className="btn btn-sm btn-primary" onClick={() => handleAction('test')} disabled={loading}>
            {loading ? <RefreshCw size={14} className="spin" /> : <Activity size={14} />}
            Test Connection
          </button>
          <button className="btn btn-sm btn-outline" onClick={reload} disabled={loading}>
            <RefreshCw size={14} /> Refresh
          </button>
        </div>
        
        {/* Config Actions */}
        <div className="action-buttons" style={{ marginTop: '12px' }}>
          <button className="btn btn-sm btn-warning" onClick={handleOpenConfigModal} disabled={loading}>
            <Edit3 size={14} /> Edit Config
          </button>
          <button className="btn btn-sm btn-info" onClick={handleSyncConfig} disabled={syncingConfig || loading}>
            {syncingConfig ? <RefreshCw size={14} className="spin" /> : <Upload size={14} />}
            Sync to All Nodes
          </button>
        </div>
        
        {testResult && (
          <div className={`alert ${testResult.success ? 'success' : 'danger'}`}>
            <AlertTriangle size={14} />
            <span>{testResult.message || (testResult.success ? 'Healthy' : 'Failed')}</span>
          </div>
        )}
      </div>

      <div className="detail-section">
        <h3>Cluster Summary</h3>
        <div className="detail-grid">
          <div className="detail-item">
            <span className="label">Status:</span>
            <StatusBadge status={clusterInfo.status} />
          </div>
          <div className="detail-item">
            <span className="label">Virtual IP:</span>
            <span className="value">{clusterInfo.virtual_ip || 'N/A'}</span>
          </div>
          <div className="detail-item">
            <span className="label">HTTP Port:</span>
            <span className="value">{clusterInfo.http_port}</span>
          </div>
          <div className="detail-item">
            <span className="label">HTTPS Port:</span>
            <span className="value">{clusterInfo.https_port || 'N/A'}</span>
          </div>
          <div className="detail-item">
            <span className="label">Load Balancing:</span>
            <span className="value">{clusterInfo.load_balance_mode || 'round_robin'}</span>
          </div>
          <div className="detail-item">
            <span className="label">SSL Enabled:</span>
            <span className="value">{clusterInfo.ssl_enabled ? 'Yes' : 'No'}</span>
          </div>
        </div>
      </div>

      <div className="detail-section">
        <h3>Nodes</h3>
        {nodes.length === 0 ? (
          <p className="text-muted">No nodes reported</p>
        ) : (
          <table className="detail-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Role</th>
                <th>Status</th>
                <th>IP</th>
                <th>Port</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map(node => (
                <tr key={node.id}>
                  <td>{node.name}</td>
                  <td>{node.role}</td>
                  <td>
                    <StatusBadge status={node.status} />
                  </td>
                  <td>{node.ip_address || 'N/A'}</td>
                  <td>{node.http_port}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {clusterInfo.upstreams?.length > 0 && (
        <div className="detail-section">
          <h3>Upstreams</h3>
          <ul className="detail-list">
            {clusterInfo.upstreams.map(upstream => (
              <li key={upstream.id}>
                <strong>{upstream.name}</strong> • {upstream.algorithm || 'round_robin'} •{' '}
                {upstream.servers?.length || 0} servers
              </li>
            ))}
          </ul>
        </div>
      )}

      {clusterInfo.server_blocks?.length > 0 && (
        <div className="detail-section">
          <h3>Server Blocks</h3>
          <ul className="detail-list">
            {clusterInfo.server_blocks.map(block => (
              <li key={block.id}>
                <Globe size={14} /> {block.server_name} • port {block.listen_port} •{' '}
                {block.ssl_enabled ? 'HTTPS' : 'HTTP'}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Edit Config Modal */}
      {showConfigModal && (
        <div className="modal-overlay inner-modal" onClick={() => setShowConfigModal(false)}>
          <div className="config-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3><Settings size={20} /> Edit Nginx Configuration</h3>
              <button className="close-btn" onClick={() => setShowConfigModal(false)}>
                <X size={20} />
              </button>
            </div>
            <div className="modal-body">
              <div className="config-info">
                <p>Edit the nginx.conf for this cluster. Changes can be saved and optionally synced to all nodes.</p>
              </div>
              <div className="form-group">
                <label>nginx.conf</label>
                <textarea 
                  className="config-editor"
                  value={configText}
                  onChange={(e) => setConfigText(e.target.value)}
                  rows={20}
                  spellCheck={false}
                />
              </div>
            </div>
            <div className="modal-footer">
              <button 
                className="btn btn-secondary" 
                onClick={() => setShowConfigModal(false)}
                disabled={savingConfig}
              >
                Cancel
              </button>
              <button 
                className="btn btn-primary" 
                onClick={() => handleSaveConfig(false)}
                disabled={savingConfig || !configText.trim()}
              >
                {savingConfig ? <RefreshCw size={14} className="spin" /> : <Download size={14} />}
                Save Only
              </button>
              <button 
                className="btn btn-success" 
                onClick={() => handleSaveConfig(true)}
                disabled={savingConfig || !configText.trim()}
              >
                {savingConfig ? <RefreshCw size={14} className="spin" /> : <Upload size={14} />}
                Save & Sync All
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const PostgresDetails = ({ details, onRefresh, onClose }) => {
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    if (window.confirm(`Delete PostgreSQL "${details.name}"? This cannot be undone!`)) {
      setDeleting(true);
      try {
        await postgresAPI.delete(details.id);
        toast.success('PostgreSQL deleted successfully');
        onClose();
        if (onRefresh) onRefresh();
      } catch (error) {
        toast.error(`Delete failed: ${error.response?.data?.message || error.message}`);
        setDeleting(false);
      }
    }
  };

  return (
    <div className="infrastructure-details">
      <div className="detail-section">
        <h3>Actions</h3>
        <div className="action-buttons">
          <button 
            className="btn btn-sm btn-danger" 
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? (
              <><RefreshCw size={14} className="spin" /> Deleting...</>
            ) : (
              <><X size={14} /> Delete</>
            )}
          </button>
        </div>
      </div>
      <div className="detail-section">
        <h3>Basic Information</h3>
        <div className="detail-grid">
          <div className="detail-item">
            <span className="label">Name:</span>
            <span className="value">{details.name}</span>
          </div>
          <div className="detail-item">
            <span className="label">Status:</span>
            <StatusBadge status={details.status} />
          </div>
          <div className="detail-item">
            <span className="label">Version:</span>
            <span className="value">{details.version}</span>
          </div>
          <div className="detail-item">
            <span className="label">Port:</span>
            <span className="value">{details.port}</span>
          </div>
          <div className="detail-item">
            <span className="label">Database:</span>
            <span className="value">{details.database_name}</span>
          </div>
          <div className="detail-item">
            <span className="label">Username:</span>
            <span className="value">{details.username}</span>
          </div>
        </div>
      </div>

      {details.connection_string && (
        <div className="detail-section">
          <h3>Connection</h3>
          <div className="connection-string">
            <code>{details.connection_string}</code>
          </div>
        </div>
      )}
    </div>
  );
};

const PostgresClusterDetails = ({ details, onRefresh, onClose }) => {
  const [actionLoading, setActionLoading] = useState({});
  const [replicationTest, setReplicationTest] = useState(null);
  const [testingReplication, setTestingReplication] = useState(false);
  const [showAddNodeModal, setShowAddNodeModal] = useState(false);
  const [newNodeName, setNewNodeName] = useState('');
  const [addingNode, setAddingNode] = useState(false);
  const clusterId = details.cluster_id || details.id;

  const handleAddNode = async () => {
    if (!newNodeName.trim()) {
      toast.error('Please enter a node name');
      return;
    }
    setAddingNode(true);
    try {
      const response = await clusterAPI.addNode(clusterId, newNodeName.trim());
      toast.success(`Node ${newNodeName} added successfully! Port: ${response.data.port}`);
      setShowAddNodeModal(false);
      setNewNodeName('');
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(`Failed to add node: ${error.response?.data?.error || error.message}`);
    } finally {
      setAddingNode(false);
    }
  };

  const handleRemoveNode = async (nodeId, nodeName, isPrimary) => {
    if (isPrimary) {
      toast.error('Cannot remove PRIMARY node. Perform failover first.');
      return;
    }
    if (!window.confirm(`Remove node ${nodeName}? This will permanently delete the node and its data.`)) {
      return;
    }
    const actionKey = `${nodeId}-remove`;
    setActionLoading(prev => ({ ...prev, [actionKey]: true }));
    try {
      await clusterAPI.removeNode(clusterId, nodeId);
      toast.success(`Node ${nodeName} removed successfully`);
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(`Failed to remove node: ${error.response?.data?.error || error.message}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [actionKey]: false }));
    }
  };

  const handleTestReplication = async () => {
    setTestingReplication(true);
    try {
      const response = await clusterAPI.testReplication(clusterId);
      setReplicationTest(response.data);
      if (response.data.all_synced) {
        toast.success('Replication test passed! All nodes are in sync.');
      } else {
        toast.error('Replication test failed! Some nodes are not synced.');
      }
    } catch (error) {
      toast.error(`Replication test failed: ${error.response?.data?.error || error.message}`);
    } finally {
      setTestingReplication(false);
    }
  };

  const handleClusterAction = async (action) => {
    setActionLoading(prev => ({ ...prev, [action]: true }));
    try {
      switch (action) {
        case 'start':
          await clusterAPI.start(clusterId);
          break;
        case 'stop':
          await clusterAPI.stop(clusterId);
          break;
        case 'restart':
          await clusterAPI.restart(clusterId);
          break;
        case 'backup':
          await clusterAPI.backup(clusterId, { type: 'full' });
          break;
        case 'scale_up':
          await clusterAPI.scale(clusterId, (details.nodes?.length || 1) + 1);
          break;
        case 'scale_down':
          if ((details.nodes?.length || 1) > 1) {
            await clusterAPI.scale(clusterId, (details.nodes?.length || 2) - 1);
          }
          break;
        default:
          break;
      }
      toast.success(`${action} successful`);
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(`${action} failed: ${error.response?.data?.message || error.message}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [action]: false }));
    }
  };

  const handleNodeAction = async (nodeId, nodeName, action, isPrimary = false) => {
    const actionKey = `${nodeId}-${action}`;
    setActionLoading(prev => ({ ...prev, [actionKey]: true }));
    try {
      switch (action) {
        case 'stop_node':
          const stopMsg = isPrimary 
            ? `Stop PRIMARY node ${nodeName}? This will trigger AUTOMATIC FAILOVER to a replica!` 
            : `Stop node ${nodeName}?`;
          if (window.confirm(stopMsg)) {
            await clusterAPI.stopNode(clusterId, nodeId);
            if (isPrimary) {
              toast.success(`Primary node stopped - automatic failover triggered! A new primary has been elected.`, { duration: 5000 });
            } else {
              toast.success(`Node ${nodeName} stopped`);
            }
          }
          break;
        case 'start_node':
          await clusterAPI.startNode(clusterId, nodeId);
          toast.success(`Node ${nodeName} started`);
          break;
        case 'failover':
          if (window.confirm(`Promote ${nodeName} to Primary? This will trigger manual failover.`)) {
            await clusterAPI.failover(clusterId, nodeId);
            toast.success(`Failover completed! ${nodeName} is now the PRIMARY.`, { duration: 5000 });
          }
          break;
        case 'reinit':
          if (window.confirm(`Reinitialize ${nodeName}? This will resync data from primary.`)) {
            await clusterAPI.patroniReinit(clusterId, { node_name: nodeName });
            toast.success(`Reinit of ${nodeName} started`);
          }
          break;
        default:
          break;
      }
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(`${action} failed: ${error.response?.data?.message || error.message}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [actionKey]: false }));
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard!');
  };

  return (
    <div className="infrastructure-details">
      {/* Cluster Actions */}
      <div className="detail-section">
        <h3>Actions</h3>
        <div className="action-buttons">
          <button 
            className="btn btn-sm btn-success" 
            onClick={() => handleClusterAction('start')}
            disabled={actionLoading['start']}
          >
            <Play size={14} /> Start
          </button>
          <button 
            className="btn btn-sm btn-warning" 
            onClick={() => handleClusterAction('stop')}
            disabled={actionLoading['stop']}
          >
            <Square size={14} /> Stop
          </button>
          <button 
            className="btn btn-sm btn-secondary" 
            onClick={() => handleClusterAction('restart')}
            disabled={actionLoading['restart']}
          >
            <RotateCcw size={14} /> Restart
          </button>
          <button 
            className="btn btn-sm btn-primary" 
            onClick={() => handleClusterAction('backup')}
            disabled={actionLoading['backup']}
          >
            <Download size={14} /> Backup
          </button>
          <button 
            className="btn btn-sm btn-info" 
            onClick={() => setShowAddNodeModal(true)}
            disabled={addingNode}
          >
            <Plus size={14} /> Add Node
          </button>
          <button 
            className="btn btn-sm btn-danger" 
            onClick={async () => {
              if (window.confirm(`Delete cluster "${details.cluster_name}"? This will remove all data and cannot be undone!`)) {
                setActionLoading(prev => ({ ...prev, delete: true }));
                try {
                  await clusterAPI.delete(clusterId);
                  toast.success('Cluster deleted successfully');
                  onClose();
                  if (onRefresh) onRefresh();
                } catch (error) {
                  toast.error(`Delete failed: ${error.response?.data?.message || error.message}`);
                  setActionLoading(prev => ({ ...prev, delete: false }));
                }
              }
            }}
            disabled={actionLoading['delete']}
          >
            {actionLoading['delete'] ? (
              <><RefreshCw size={14} className="spin" /> Deleting...</>
            ) : (
              <><X size={14} /> Delete Cluster</>
            )}
          </button>
        </div>
      </div>

      <div className="detail-section">
        <h3>Cluster Information</h3>
        <div className="detail-grid">
          <div className="detail-item">
            <span className="label">Cluster Name:</span>
            <span className="value">{details.cluster_name}</span>
          </div>
          <div className="detail-item">
            <span className="label">Status:</span>
            <StatusBadge status={details.status} />
          </div>
          <div className="detail-item">
            <span className="label">PostgreSQL Version:</span>
            <span className="value">{details.postgres_version}</span>
          </div>
          <div className="detail-item">
            <span className="label">Node Count:</span>
            <span className="value">{details.node_count || details.nodes?.length || 0}</span>
          </div>
          <div className="detail-item">
            <span className="label">Replication Mode:</span>
            <span className="value">{details.replication_mode}</span>
          </div>
          <div className="detail-item">
            <span className="label">HAProxy Port:</span>
            <span className="value">{details.haproxy_port}</span>
          </div>
        </div>
      </div>

      {details.write_endpoint && (
        <div className="detail-section">
          <h3>Connection Endpoints</h3>
          <div className="endpoints-list">
            <div className="endpoint-item">
              <span className="label">Write (Primary):</span>
              <code className="value">{details.write_endpoint.host}:{details.write_endpoint.port}</code>
              <button className="btn-copy" onClick={() => copyToClipboard(`${details.write_endpoint.host}:${details.write_endpoint.port}`)}>
                <Copy size={12} />
              </button>
            </div>
            {details.read_endpoints?.length > 0 && (
              <div className="endpoint-item">
                <span className="label">Read (Replicas):</span>
                <span className="value">{details.read_endpoints.length} available</span>
              </div>
            )}
          </div>
        </div>
      )}

      {details.nodes && details.nodes.length > 0 && (
        <div className="detail-section">
          <h3>Cluster Nodes ({details.nodes.length})</h3>
          <div className="nodes-table-container">
            <table className="nodes-table">
              <thead>
                <tr>
                  <th>Node Name</th>
                  <th>Role</th>
                  <th>Status</th>
                  <th>Health</th>
                  <th>Replication Lag</th>
                  <th>Container ID</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {details.nodes.map((node, idx) => {
                  const isPrimary = node.role === 'leader' || node.role === 'primary';
                  return (
                    <tr key={idx} className={isPrimary ? 'row-primary' : ''}>
                      <td>
                        <div className="node-name-cell">
                          <Database size={14} />
                          <span>{node.node_name}</span>
                        </div>
                      </td>
                      <td>
                        <span className={`role-badge ${isPrimary ? 'primary' : 'replica'}`}>
                          {isPrimary ? <><Crown size={12} /> Primary</> : 'Replica'}
                        </span>
                      </td>
                      <td><StatusBadge status={node.status} /></td>
                      <td>
                        <span className={`health-indicator ${node.is_healthy ? 'healthy' : 'unhealthy'}`}>
                          {node.is_healthy ? '✓ Healthy' : '✗ Unhealthy'}
                        </span>
                      </td>
                      <td>
                        {node.replication_delay !== undefined && node.replication_delay !== null 
                          ? `${node.replication_delay} bytes` 
                          : '-'}
                      </td>
                      <td>
                        <code className="container-id">{node.container_id?.substring(0, 12)}</code>
                      </td>
                      <td>
                        <div className="node-actions">
                          {/* Stop/Start Node buttons */}
                          {node.is_healthy ? (
                            <button 
                              className={`btn btn-xs ${isPrimary ? 'btn-danger' : 'btn-warning'}`}
                              onClick={() => handleNodeAction(node.node_id, node.node_name, 'stop_node', isPrimary)}
                              disabled={actionLoading[`${node.node_id}-stop_node`]}
                              title={isPrimary ? "Stop Primary (triggers automatic failover)" : "Stop Node"}
                            >
                              <Square size={12} /> {isPrimary ? 'Stop (Failover)' : 'Stop'}
                            </button>
                          ) : (
                            <button 
                              className="btn btn-xs btn-success"
                              onClick={() => handleNodeAction(node.node_id, node.node_name, 'start_node')}
                              disabled={actionLoading[`${node.node_id}-start_node`]}
                              title="Start Node"
                            >
                              <Play size={12} /> Start
                            </button>
                          )}
                          {/* Remove Node button - only for replicas */}
                          {!isPrimary && (
                            <button 
                              className="btn btn-xs btn-danger"
                              onClick={() => handleRemoveNode(node.node_id, node.node_name, isPrimary)}
                              disabled={actionLoading[`${node.node_id}-remove`]}
                              title="Remove this node from cluster"
                            >
                              {actionLoading[`${node.node_id}-remove`] ? (
                                <RefreshCw size={12} className="spin" />
                              ) : (
                                <><Minus size={12} /> Remove</>
                              )}
                            </button>
                          )}
                          {/* Info button */}
                          <button 
                            className="btn btn-xs btn-info"
                            onClick={() => alert(`Node Details:\n\nName: ${node.node_name}\nRole: ${node.role}\nStatus: ${node.status}\nHealthy: ${node.is_healthy}\nContainer: ${node.container_id}\nReplication Lag: ${node.replication_delay || 0} bytes`)}
                            title="View Node Details"
                          >
                            <Database size={12} /> Info
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Replication Test Section */}
      <div className="detail-section">
        <h3>Replication Test</h3>
        <div className="replication-test-section">
          <button 
            className="btn btn-sm btn-primary"
            onClick={handleTestReplication}
            disabled={testingReplication}
          >
            {testingReplication ? (
              <><RefreshCw size={14} className="spin" /> Testing...</>
            ) : (
              <><Database size={14} /> Test Replication</>
            )}
          </button>
          
          {replicationTest && (
            <div className="replication-result">
              <div className={`sync-status ${replicationTest.all_synced ? 'synced' : 'not-synced'}`}>
                {replicationTest.all_synced ? '✓ All nodes in sync' : '✗ Replication issue detected'}
              </div>
              <table className="nodes-table mini">
                <thead>
                  <tr>
                    <th>Node</th>
                    <th>Role</th>
                    <th>Has Data</th>
                    <th>Row Count</th>
                  </tr>
                </thead>
                <tbody>
                  {replicationTest.node_results?.map((node, idx) => (
                    <tr key={idx} className={node.role === 'primary' ? 'row-primary' : ''}>
                      <td>{node.node_name}</td>
                      <td><span className={`role-badge ${node.role}`}>{node.role}</span></td>
                      <td>{node.has_data ? '✓' : '✗'}</td>
                      <td>{node.row_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <small>Test performed at: {replicationTest.test_timestamp}</small>
            </div>
          )}
        </div>
      </div>

      {/* Add Node Modal */}
      {showAddNodeModal && (
        <div className="modal-overlay inner-modal" onClick={() => setShowAddNodeModal(false)}>
          <div className="add-node-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3><Plus size={20} /> Add New Node</h3>
              <button className="close-btn" onClick={() => setShowAddNodeModal(false)}>
                <X size={20} />
              </button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label>Node Name</label>
                <input 
                  type="text" 
                  className="form-control"
                  value={newNodeName}
                  onChange={(e) => setNewNodeName(e.target.value)}
                  placeholder="e.g., node-4"
                  disabled={addingNode}
                />
                <small className="help-text">The new node will be added as a replica and automatically sync data from the primary.</small>
              </div>
            </div>
            <div className="modal-footer">
              <button 
                className="btn btn-secondary" 
                onClick={() => setShowAddNodeModal(false)}
                disabled={addingNode}
              >
                Cancel
              </button>
              <button 
                className="btn btn-primary" 
                onClick={handleAddNode}
                disabled={addingNode || !newNodeName.trim()}
              >
                {addingNode ? (
                  <><RefreshCw size={14} className="spin" /> Adding...</>
                ) : (
                  <><Plus size={14} /> Add Node</>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const DinDEnvironmentDetails = ({ details, onRefresh, onClose }) => {
  const [deleting, setDeleting] = useState(false);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [stats, setStats] = useState(null);
  const [containers, setContainers] = useState([]);
  const [images, setImages] = useState([]);
  const [activeTab, setActiveTab] = useState('info');
  const [command, setCommand] = useState('');
  const [commandOutput, setCommandOutput] = useState('');
  const [executing, setExecuting] = useState(false);
  const [pullImage, setPullImage] = useState('');
  const [pulling, setPulling] = useState(false);

  useEffect(() => {
    if (details?.id) {
      loadStats();
      loadContainers();
      loadImages();
    }
  }, [details]);

  const loadStats = async () => {
    try {
      const response = await dinDAPI.getStats(details.id);
      if (response.data?.success) {
        setStats(response.data.data);
      }
    } catch (error) {
      console.error('Error loading stats:', error);
    }
  };

  const loadContainers = async () => {
    try {
      const response = await dinDAPI.listContainers(details.id);
      if (response.data?.success) {
        setContainers(response.data.data || []);
      }
    } catch (error) {
      console.error('Error loading containers:', error);
    }
  };

  const loadImages = async () => {
    try {
      const response = await dinDAPI.listImages(details.id);
      if (response.data?.success) {
        setImages(response.data.data || []);
      }
    } catch (error) {
      console.error('Error loading images:', error);
    }
  };

  const handleDelete = async () => {
    if (window.confirm(`Delete DinD Environment "${details.name}"? All containers and images inside will be lost!`)) {
      setDeleting(true);
      try {
        await dinDAPI.deleteEnvironment(details.id);
        toast.success('DinD Environment deleted successfully');
        onClose();
        if (onRefresh) onRefresh();
      } catch (error) {
        toast.error(`Delete failed: ${error.response?.data?.message || error.message}`);
        setDeleting(false);
      }
    }
  };

  const handleStart = async () => {
    setStarting(true);
    try {
      await dinDAPI.startEnvironment(details.id);
      toast.success('Environment started successfully');
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(`Start failed: ${error.response?.data?.message || error.message}`);
    } finally {
      setStarting(false);
    }
  };

  const handleStop = async () => {
    setStopping(true);
    try {
      await dinDAPI.stopEnvironment(details.id);
      toast.success('Environment stopped successfully');
      if (onRefresh) onRefresh();
    } catch (error) {
      toast.error(`Stop failed: ${error.response?.data?.message || error.message}`);
    } finally {
      setStopping(false);
    }
  };

  const handleExecCommand = async () => {
    if (!command.trim()) return;
    setExecuting(true);
    setCommandOutput('');
    try {
      const response = await dinDAPI.execCommand(details.id, command, 60);
      if (response.data?.success) {
        setCommandOutput(response.data.data?.output || 'Command executed successfully');
      } else {
        setCommandOutput(`Error: ${response.data?.message || 'Unknown error'}`);
      }
    } catch (error) {
      setCommandOutput(`Error: ${error.response?.data?.message || error.message}`);
    } finally {
      setExecuting(false);
    }
  };

  const handlePullImage = async () => {
    if (!pullImage.trim()) return;
    setPulling(true);
    try {
      await dinDAPI.pullImage(details.id, pullImage);
      toast.success(`Image ${pullImage} pulled successfully`);
      setPullImage('');
      loadImages();
    } catch (error) {
      toast.error(`Pull failed: ${error.response?.data?.message || error.message}`);
    } finally {
      setPulling(false);
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  };

  const formatBytes = (bytes) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (dateStr) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleString();
  };

  return (
    <div className="infrastructure-details dind-details">
      {/* Actions */}
      <div className="detail-section">
        <h3>Actions</h3>
        <div className="action-buttons">
          {details.status === 'stopped' ? (
            <button 
              className="btn btn-sm btn-success" 
              onClick={handleStart}
              disabled={starting}
            >
              {starting ? <><RefreshCw size={14} className="spin" /> Starting...</> : <><Play size={14} /> Start</>}
            </button>
          ) : (
            <button 
              className="btn btn-sm btn-warning" 
              onClick={handleStop}
              disabled={stopping}
            >
              {stopping ? <><RefreshCw size={14} className="spin" /> Stopping...</> : <><Square size={14} /> Stop</>}
            </button>
          )}
          <button 
            className="btn btn-sm btn-info" 
            onClick={() => { loadStats(); loadContainers(); loadImages(); }}
          >
            <RefreshCw size={14} /> Refresh
          </button>
          <button 
            className="btn btn-sm btn-danger" 
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? <><RefreshCw size={14} className="spin" /> Deleting...</> : <><X size={14} /> Delete</>}
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="detail-tabs">
        <button className={`tab-btn ${activeTab === 'info' ? 'active' : ''}`} onClick={() => setActiveTab('info')}>
          Info
        </button>
        <button className={`tab-btn ${activeTab === 'terminal' ? 'active' : ''}`} onClick={() => setActiveTab('terminal')}>
          Terminal
        </button>
        <button className={`tab-btn ${activeTab === 'containers' ? 'active' : ''}`} onClick={() => setActiveTab('containers')}>
          Containers ({containers.length})
        </button>
        <button className={`tab-btn ${activeTab === 'images' ? 'active' : ''}`} onClick={() => setActiveTab('images')}>
          Images ({images.length})
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === 'info' && (
        <>
          {/* Basic Information */}
          <div className="detail-section">
            <h3>Basic Information</h3>
            <div className="detail-grid">
              <div className="detail-item">
                <span className="label">Environment ID:</span>
                <span className="value code clickable" onClick={() => copyToClipboard(details.id)}>
                  {details.id?.substring(0, 8)}... <Copy size={12} />
                </span>
              </div>
              <div className="detail-item">
                <span className="label">Name:</span>
                <span className="value">{details.name}</span>
              </div>
              <div className="detail-item">
                <span className="label">Status:</span>
                <StatusBadge status={details.status} />
              </div>
              <div className="detail-item">
                <span className="label">Resource Plan:</span>
                <span className="value badge">{details.resource_plan}</span>
              </div>
            </div>
          </div>

          {/* Connection Info */}
          <div className="detail-section">
            <h3>Connection</h3>
            <div className="detail-grid">
              <div className="detail-item">
                <span className="label">Docker Host:</span>
                <span className="value code clickable" onClick={() => copyToClipboard(details.docker_host)}>
                  {details.docker_host} <Copy size={12} />
                </span>
              </div>
              <div className="detail-item">
                <span className="label">IP Address:</span>
                <span className="value code">{details.ip_address}</span>
              </div>
              <div className="detail-item">
                <span className="label">Container ID:</span>
                <span className="value code clickable" onClick={() => copyToClipboard(details.container_id)}>
                  {details.container_id?.substring(0, 12)}... <Copy size={12} />
                </span>
              </div>
            </div>
          </div>

          {/* Resource Limits */}
          <div className="detail-section">
            <h3>Resource Limits</h3>
            <div className="detail-grid">
              <div className="detail-item">
                <span className="label">CPU Limit:</span>
                <span className="value">{details.cpu_limit} cores</span>
              </div>
              <div className="detail-item">
                <span className="label">Memory Limit:</span>
                <span className="value">{details.memory_limit}</span>
              </div>
              <div className="detail-item">
                <span className="label">Auto Cleanup:</span>
                <span className="value">{details.auto_cleanup ? 'Yes' : 'No'}</span>
              </div>
              <div className="detail-item">
                <span className="label">TTL:</span>
                <span className="value">{details.ttl_hours > 0 ? `${details.ttl_hours} hours` : 'No expiration'}</span>
              </div>
            </div>
          </div>

          {/* Statistics */}
          {stats && (
            <div className="detail-section">
              <h3>Statistics</h3>
              <div className="detail-grid">
                <div className="detail-item">
                  <span className="label">CPU Usage:</span>
                  <span className="value">{stats.cpu_usage_percent?.toFixed(1) || 0}%</span>
                </div>
                <div className="detail-item">
                  <span className="label">Memory Usage:</span>
                  <span className="value">
                    {formatBytes(stats.memory_usage_bytes)} / {formatBytes(stats.memory_limit_bytes)}
                    ({stats.memory_usage_percent?.toFixed(1) || 0}%)
                  </span>
                </div>
                <div className="detail-item">
                  <span className="label">Network RX:</span>
                  <span className="value">{formatBytes(stats.network_rx_bytes)}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Network TX:</span>
                  <span className="value">{formatBytes(stats.network_tx_bytes)}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Containers:</span>
                  <span className="value">{stats.container_count || 0}</span>
                </div>
                <div className="detail-item">
                  <span className="label">Images:</span>
                  <span className="value">{stats.image_count || 0}</span>
                </div>
              </div>
            </div>
          )}

          {/* Timestamps */}
          <div className="detail-section">
            <h3>Timestamps</h3>
            <div className="detail-grid">
              <div className="detail-item">
                <span className="label">Created:</span>
                <span className="value">{formatDate(details.created_at)}</span>
              </div>
              <div className="detail-item">
                <span className="label">Updated:</span>
                <span className="value">{formatDate(details.updated_at)}</span>
              </div>
              {details.expires_at && (
                <div className="detail-item">
                  <span className="label">Expires:</span>
                  <span className="value">{formatDate(details.expires_at)}</span>
                </div>
              )}
            </div>
          </div>

          {details.description && (
            <div className="detail-section">
              <h3>Description</h3>
              <p>{details.description}</p>
            </div>
          )}
        </>
      )}

      {activeTab === 'terminal' && (
        <div className="detail-section terminal-section">
          <h3>Docker Terminal</h3>
          <div className="terminal-input">
            <input
              type="text"
              placeholder="Enter docker command (e.g., docker ps, docker images)"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleExecCommand()}
              disabled={executing || details.status !== 'running'}
            />
            <button 
              className="btn btn-sm btn-primary"
              onClick={handleExecCommand}
              disabled={executing || !command.trim() || details.status !== 'running'}
            >
              {executing ? <RefreshCw size={14} className="spin" /> : <Play size={14} />}
              Execute
            </button>
          </div>
          {commandOutput && (
            <pre className="terminal-output">{commandOutput}</pre>
          )}
          
          <div className="pull-image-section">
            <h4>Pull Image</h4>
            <div className="terminal-input">
              <input
                type="text"
                placeholder="Image name (e.g., nginx:latest, redis:7)"
                value={pullImage}
                onChange={(e) => setPullImage(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handlePullImage()}
                disabled={pulling || details.status !== 'running'}
              />
              <button 
                className="btn btn-sm btn-success"
                onClick={handlePullImage}
                disabled={pulling || !pullImage.trim() || details.status !== 'running'}
              >
                {pulling ? <RefreshCw size={14} className="spin" /> : <Download size={14} />}
                Pull
              </button>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'containers' && (
        <div className="detail-section">
          <h3>Containers Inside DinD</h3>
          {containers.length === 0 ? (
            <p className="empty-message">No containers running</p>
          ) : (
            <div className="table-container">
              <table className="detail-table">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Image</th>
                    <th>Status</th>
                    <th>Names</th>
                  </tr>
                </thead>
                <tbody>
                  {containers.map((container, idx) => (
                    <tr key={idx}>
                      <td className="code">{container.id?.substring(0, 12)}</td>
                      <td>{container.image}</td>
                      <td><StatusBadge status={container.state} /></td>
                      <td>{container.names?.join(', ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {activeTab === 'images' && (
        <div className="detail-section">
          <h3>Images Inside DinD</h3>
          {images.length === 0 ? (
            <p className="empty-message">No images available</p>
          ) : (
            <div className="table-container">
              <table className="detail-table">
                <thead>
                  <tr>
                    <th>Repository</th>
                    <th>Tag</th>
                    <th>Size</th>
                    <th>ID</th>
                  </tr>
                </thead>
                <tbody>
                  {images.map((image, idx) => (
                    <tr key={idx}>
                      <td>{image.repository || '<none>'}</td>
                      <td>{image.tag || '<none>'}</td>
                      <td>{formatBytes(image.size)}</td>
                      <td className="code">{image.id?.substring(0, 12)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default InfrastructureDetailModal;

