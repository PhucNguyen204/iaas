import React, { useState, useEffect } from 'react';
import { clusterAPI } from '../api';

function PostgresCluster() {
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [showDetailsModal, setShowDetailsModal] = useState(false);

  const loadClusters = async () => {
    setLoading(true);
    try {
      // Since there's no getAll endpoint, we'll maintain a list in state
      // In production, you'd fetch from backend
      setLoading(false);
    } catch (error) {
      console.error('Error loading clusters:', error);
      setLoading(false);
    }
  };

  useEffect(() => {
    loadClusters();
  }, []);

  const handleCreateCluster = async (formData) => {
    try {
      console.log('Creating cluster with data:', formData);
      const response = await clusterAPI.create(formData);
      console.log('Create response:', response);
      if (response.data.success) {
        alert('Cluster created successfully!');
        setShowCreateModal(false);
        // Add to clusters list
        setClusters([...clusters, response.data.data]);
      }
    } catch (error) {
      console.error('Create cluster error:', error);
      alert('Error creating cluster: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleViewDetails = async (clusterId) => {
    try {
      const response = await clusterAPI.getById(clusterId);
      if (response.data.success) {
        setSelectedCluster(response.data.data);
        setShowDetailsModal(true);
      }
    } catch (error) {
      alert('Error fetching cluster details: ' + error.message);
    }
  };

  const handleOperation = async (clusterId, operation) => {
    try {
      let response;
      switch (operation) {
        case 'start':
          response = await clusterAPI.start(clusterId);
          break;
        case 'stop':
          response = await clusterAPI.stop(clusterId);
          break;
        case 'restart':
          response = await clusterAPI.restart(clusterId);
          break;
        case 'delete':
          if (!window.confirm('Are you sure you want to delete this cluster?')) return;
          response = await clusterAPI.delete(clusterId);
          setClusters(clusters.filter(c => c.id !== clusterId));
          break;
        default:
          return;
      }
      if (response.data.success) {
        alert(`Cluster ${operation} successful!`);
        loadClusters();
      }
    } catch (error) {
      alert(`Error ${operation} cluster: ` + error.message);
    }
  };

  return (
    <div>
      <button className="create-button" onClick={() => {
        console.log('Create button clicked');
        setShowCreateModal(true);
      }}>
        + Create PostgreSQL Cluster
      </button>

      {loading ? (
        <div className="loading">Loading clusters...</div>
      ) : clusters.length === 0 ? (
        <div className="empty-state">
          <h3>No PostgreSQL clusters yet</h3>
          <p>Create your first cluster to get started</p>
        </div>
      ) : (
        <div className="cards-grid">
          {clusters.map((cluster) => (
            <div key={cluster.id} className="card">
              <h3>{cluster.cluster_name}</h3>
              <div className="card-info">
                <strong>Status:</strong>{' '}
                <span className={`status-badge ${cluster.status}`}>
                  {cluster.status}
                </span>
              </div>
              <div className="card-info">
                <strong>Version:</strong> PostgreSQL {cluster.version}
              </div>
              <div className="card-info">
                <strong>Nodes:</strong> {cluster.node_count}
              </div>
              <div className="card-info">
                <strong>Replication:</strong> {cluster.replication_mode}
              </div>
              
              <div className="card-actions">
                <button className="btn-sm btn-info" onClick={() => handleViewDetails(cluster.id)}>
                  Details
                </button>
                <button className="btn-sm btn-success" onClick={() => handleOperation(cluster.id, 'start')}>
                  Start
                </button>
                <button className="btn-sm btn-warning" onClick={() => handleOperation(cluster.id, 'stop')}>
                  Stop
                </button>
                <button className="btn-sm btn-primary" onClick={() => handleOperation(cluster.id, 'restart')}>
                  Restart
                </button>
                <button className="btn-sm btn-danger" onClick={() => handleOperation(cluster.id, 'delete')}>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreateModal && (
        <CreateClusterModal
          onClose={() => setShowCreateModal(false)}
          onCreate={handleCreateCluster}
        />
      )}

      {showDetailsModal && selectedCluster && (
        <ClusterDetailsModal
          cluster={selectedCluster}
          onClose={() => setShowDetailsModal(false)}
        />
      )}
    </div>
  );
}

function CreateClusterModal({ onClose, onCreate }) {
  const [formData, setFormData] = useState({
    cluster_name: '',
    postgres_version: '15',
    node_count: 3,
    cpu_per_node: 512,
    memory_per_node: 536870912, // 512MB
    storage_per_node: 10737418240, // 10GB
    postgres_password: '',
    replication_mode: 'async'
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onCreate(formData);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Create PostgreSQL Cluster</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Cluster Name</label>
            <input
              type="text"
              value={formData.cluster_name}
              onChange={(e) => setFormData({...formData, cluster_name: e.target.value})}
              required
            />
          </div>

          <div className="form-group">
            <label>PostgreSQL Version</label>
            <select
              value={formData.postgres_version}
              onChange={(e) => setFormData({...formData, postgres_version: e.target.value})}
            >
              <option value="15">15</option>
              <option value="14">14</option>
              <option value="13">13</option>
            </select>
          </div>

          <div className="form-group">
            <label>Number of Nodes</label>
            <input
              type="number"
              min="1"
              max="10"
              value={formData.node_count}
              onChange={(e) => setFormData({...formData, node_count: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>CPU per Node (millicores)</label>
            <input
              type="number"
              value={formData.cpu_per_node}
              onChange={(e) => setFormData({...formData, cpu_per_node: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Memory per Node (bytes)</label>
            <input
              type="number"
              value={formData.memory_per_node}
              onChange={(e) => setFormData({...formData, memory_per_node: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Storage per Node (bytes)</label>
            <input
              type="number"
              value={formData.storage_per_node}
              onChange={(e) => setFormData({...formData, storage_per_node: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>PostgreSQL Password</label>
            <input
              type="password"
              value={formData.postgres_password}
              onChange={(e) => setFormData({...formData, postgres_password: e.target.value})}
              required
            />
          </div>

          <div className="form-group">
            <label>Replication Mode</label>
            <select
              value={formData.replication_mode}
              onChange={(e) => setFormData({...formData, replication_mode: e.target.value})}
            >
              <option value="async">Asynchronous</option>
              <option value="sync">Synchronous</option>
            </select>
          </div>

          <div className="modal-actions">
            <button type="button" className="btn btn-danger" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn btn-success">
              Create Cluster
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function ClusterDetailsModal({ cluster, onClose }) {
  const [endpoints, setEndpoints] = useState(null);
  const [replication, setReplication] = useState(null);

  useEffect(() => {
    loadEndpoints();
    loadReplication();
  }, [cluster.id]);

  const loadEndpoints = async () => {
    try {
      const response = await clusterAPI.getEndpoints(cluster.id);
      if (response.data.success) {
        setEndpoints(response.data.data);
      }
    } catch (error) {
      console.error('Error loading endpoints:', error);
    }
  };

  const loadReplication = async () => {
    try {
      const response = await clusterAPI.getReplication(cluster.id);
      if (response.data.success) {
        setReplication(response.data.data);
      }
    } catch (error) {
      console.error('Error loading replication:', error);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>{cluster.cluster_name}</h2>
        
        <div className="card-info">
          <strong>Status:</strong> <span className={`status-badge ${cluster.status}`}>{cluster.status}</span>
        </div>
        <div className="card-info">
          <strong>Version:</strong> PostgreSQL {cluster.version}
        </div>
        <div className="card-info">
          <strong>Nodes:</strong> {cluster.node_count}
        </div>
        <div className="card-info">
          <strong>Replication Mode:</strong> {cluster.replication_mode}
        </div>

        {endpoints && (
          <div className="connection-box">
            <h3>Connection Endpoints</h3>
            {endpoints.write_endpoint && (
              <div className="endpoint">
                <strong>Write (Primary):</strong> {endpoints.write_endpoint.host}:{endpoints.write_endpoint.port}
              </div>
            )}
            {endpoints.read_endpoints && endpoints.read_endpoints.length > 0 && (
              <div>
                <strong>Read (Replicas):</strong>
                {endpoints.read_endpoints.map((ep, idx) => (
                  <div key={idx} className="endpoint">
                    • {ep.host}:{ep.port}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {cluster.nodes && cluster.nodes.length > 0 && (
          <div className="node-list">
            <h3>Nodes</h3>
            {cluster.nodes.map((node, idx) => (
              <div key={idx} className="node-item">
                <span className="node-role">{node.role}</span>
                <span className="node-port">Port: {node.port}</span>
                <span className={`status-badge ${node.is_healthy ? 'running' : 'stopped'}`}>
                  {node.is_healthy ? 'Healthy' : 'Unhealthy'}
                </span>
              </div>
            ))}
          </div>
        )}

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}

export default PostgresCluster;
