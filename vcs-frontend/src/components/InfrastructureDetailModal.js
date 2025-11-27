import React, { useState, useEffect } from 'react';
import { X, Database, RefreshCw, Play, Square, RotateCcw, Crown, Plus, Minus, Copy, Download } from 'lucide-react';
import { postgresAPI, nginxAPI, dockerAPI, clusterAPI } from '../api';
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
        
        case 'DOCKER_SERVICE':
          response = await dockerAPI.getById(resource.infrastructure_id);
          setDetails(response.data?.data || response.data);
          break;
        
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
    case 'POSTGRES_INSTANCE':
      return <PostgresDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    case 'POSTGRES_CLUSTER':
      return <PostgresClusterDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
    case 'DOCKER_SERVICE':
      return <DockerServiceDetails details={details} onRefresh={onRefresh} onClose={onClose} />;
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

const DockerServiceDetails = ({ details, onRefresh, onClose }) => {
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    if (window.confirm(`Delete Docker Service "${details.name}"? This cannot be undone!`)) {
      setDeleting(true);
      try {
        await dockerAPI.delete(details.id);
        toast.success('Docker Service deleted successfully');
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
            <span className="label">Image:</span>
            <span className="value">{details.image}</span>
          </div>
          <div className="detail-item">
            <span className="label">Container ID:</span>
            <span className="value code">{details.container_id?.substring(0, 12)}</span>
          </div>
        </div>
      </div>

      {details.ports && details.ports.length > 0 && (
        <div className="detail-section">
          <h3>Ports</h3>
          <ul>
            {details.ports.map((port, idx) => (
              <li key={idx}>{port.container_port}:{port.host_port} ({port.protocol})</li>
            ))}
          </ul>
        </div>
      )}

      {details.env_vars && details.env_vars.length > 0 && (
        <div className="detail-section">
          <h3>Environment Variables</h3>
          <div className="env-vars-list">
            {details.env_vars.map((env, idx) => (
              <div key={idx} className="env-var-item">
                <span className="env-key">{env.key}:</span>
                <span className="env-value">{env.is_secret ? '****' : env.value}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default InfrastructureDetailModal;

