import React, { useState } from 'react';
import { X, Plus, AlertCircle, CheckCircle, Copy } from 'lucide-react';
import { nginxClusterAPI } from '../../api';
import toast from 'react-hot-toast';
import './CreateNginxClusterModal.css';

const CreateNginxClusterModal = ({ isOpen, onClose, onSuccess }) => {
  const [formData, setFormData] = useState({
    cluster_name: '',
    node_count: 2,
    http_port: 8080,
    https_port: 8443,
    load_balance_mode: 'round_robin',
    virtual_ip: '',
    worker_connections: 2048,
    worker_processes: 2,
    ssl_enabled: false,
    gzip_enabled: true,
    health_check_enabled: true,
    health_check_path: '/health',
    rate_limit_enabled: false,
    cache_enabled: false,
  });

  const [isCreating, setIsCreating] = useState(false);
  const [createdCluster, setCreatedCluster] = useState(null);

  if (!isOpen) return null;

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard!');
  };

  const handleClose = () => {
    setCreatedCluster(null);
    setFormData({
      cluster_name: '',
      node_count: 2,
      http_port: 8080,
      https_port: 8443,
      load_balance_mode: 'round_robin',
      virtual_ip: '',
      worker_connections: 2048,
      worker_processes: 2,
      ssl_enabled: false,
      gzip_enabled: true,
      health_check_enabled: true,
      health_check_path: '/health',
      rate_limit_enabled: false,
      cache_enabled: false,
    });
    onClose();
  };

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!formData.cluster_name.trim()) {
      toast.error('Cluster name is required');
      return;
    }

    if (formData.node_count < 2) {
      toast.error('Minimum 2 nodes required for HA cluster');
      return;
    }

    setIsCreating(true);

    try {
      const response = await nginxClusterAPI.create(formData);
      
      if (response.data.success) {
        toast.success('Nginx Cluster created successfully!');
        setCreatedCluster(response.data.data);
        onSuccess && onSuccess(response.data.data);
      } else {
        toast.error(response.data.message || 'Failed to create cluster');
      }
    } catch (error) {
      console.error('Error creating nginx cluster:', error);
      toast.error(error.response?.data?.message || 'Failed to create cluster');
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-container nginx-cluster-modal">
        <div className="modal-header">
          <h2>{createdCluster ? 'Cluster Created Successfully!' : 'Create Nginx HA Cluster'}</h2>
          <button className="close-btn" onClick={handleClose}>
            <X size={24} />
          </button>
        </div>

        {createdCluster ? (
          /* Success View - Show created cluster info */
          <div className="modal-body">
            <div className="success-banner">
              <CheckCircle size={48} color="#10b981" />
              <h3>Nginx Cluster Created!</h3>
            </div>

            <div className="created-cluster-info">
              <div className="info-item">
                <label>Cluster ID:</label>
                <div className="info-value-with-copy">
                  <code className="cluster-id">{createdCluster.id}</code>
                  <button 
                    className="copy-btn" 
                    onClick={() => copyToClipboard(createdCluster.id)}
                    title="Copy ID"
                  >
                    <Copy size={16} />
                  </button>
                </div>
              </div>

              <div className="info-item">
                <label>Cluster Name:</label>
                <span>{createdCluster.cluster_name}</span>
              </div>

              <div className="info-item">
                <label>Infrastructure ID:</label>
                <div className="info-value-with-copy">
                  <code>{createdCluster.infrastructure_id}</code>
                  <button 
                    className="copy-btn" 
                    onClick={() => copyToClipboard(createdCluster.infrastructure_id)}
                    title="Copy Infrastructure ID"
                  >
                    <Copy size={16} />
                  </button>
                </div>
              </div>

              <div className="info-item">
                <label>Status:</label>
                <span className={`status-badge status-${createdCluster.status}`}>
                  {createdCluster.status}
                </span>
              </div>

              <div className="info-item">
                <label>Node Count:</label>
                <span>{createdCluster.node_count} nodes</span>
              </div>

              <div className="info-item">
                <label>HTTP Port:</label>
                <span>{createdCluster.http_port}</span>
              </div>

              {createdCluster.https_port > 0 && (
                <div className="info-item">
                  <label>HTTPS Port:</label>
                  <span>{createdCluster.https_port}</span>
                </div>
              )}

              {createdCluster.endpoints && (
                <div className="info-item">
                  <label>HTTP URL:</label>
                  <div className="info-value-with-copy">
                    <code>{createdCluster.endpoints.http_url}</code>
                    <button 
                      className="copy-btn" 
                      onClick={() => copyToClipboard(createdCluster.endpoints.http_url)}
                      title="Copy URL"
                    >
                      <Copy size={16} />
                    </button>
                  </div>
                </div>
              )}

              {createdCluster.nodes && createdCluster.nodes.length > 0 && (
                <div className="info-item nodes-list">
                  <label>Nodes:</label>
                  <div className="nodes-container">
                    {createdCluster.nodes.map((node, index) => (
                      <div key={node.id || index} className="node-item">
                        <span className="node-name">{node.name}</span>
                        <span className={`node-role ${node.role}`}>{node.role}</span>
                        <span className={`node-status ${node.status}`}>{node.status}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="modal-footer">
              <button
                type="button"
                className="btn btn-primary"
                onClick={handleClose}
              >
                Done
              </button>
            </div>
          </div>
        ) : (
        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div className="info-banner">
              <AlertCircle size={16} />
              <span>High Availability Nginx Cluster with Keepalived + VRRP</span>
            </div>

            {/* Basic Configuration */}
            <div className="config-section">
              <h3>Basic Configuration</h3>
              
              <div className="form-row">
                <div className="form-group">
                  <label>Cluster Name *</label>
                  <input
                    type="text"
                    name="cluster_name"
                    value={formData.cluster_name}
                    onChange={handleInputChange}
                    placeholder="my-nginx-cluster"
                    required
                  />
                </div>

                <div className="form-group">
                  <label>Node Count *</label>
                  <input
                    type="number"
                    name="node_count"
                    value={formData.node_count}
                    onChange={handleInputChange}
                    min="2"
                    max="10"
                    required
                  />
                  <small>Minimum 2 nodes for HA</small>
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>HTTP Port *</label>
                  <input
                    type="number"
                    name="http_port"
                    value={formData.http_port}
                    onChange={handleInputChange}
                    min="1024"
                    max="65535"
                    required
                  />
                </div>

                <div className="form-group">
                  <label>HTTPS Port</label>
                  <input
                    type="number"
                    name="https_port"
                    value={formData.https_port}
                    onChange={handleInputChange}
                    min="1024"
                    max="65535"
                  />
                </div>
              </div>
            </div>

            {/* Network Configuration */}
            <div className="config-section">
              <h3>Network & Load Balancing</h3>
              
              <div className="form-row">
                <div className="form-group">
                  <label>Virtual IP (Optional)</label>
                  <input
                    type="text"
                    name="virtual_ip"
                    value={formData.virtual_ip}
                    onChange={handleInputChange}
                    placeholder="192.168.0.100"
                  />
                  <small>Keepalived VIP for failover</small>
                </div>

                <div className="form-group">
                  <label>Load Balance Mode</label>
                  <select
                    name="load_balance_mode"
                    value={formData.load_balance_mode}
                    onChange={handleInputChange}
                  >
                    <option value="round_robin">Round Robin</option>
                    <option value="least_conn">Least Connections</option>
                    <option value="ip_hash">IP Hash</option>
                    <option value="random">Random</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Performance Configuration */}
            <div className="config-section">
              <h3>Performance</h3>
              
              <div className="form-row">
                <div className="form-group">
                  <label>Worker Processes</label>
                  <input
                    type="number"
                    name="worker_processes"
                    value={formData.worker_processes}
                    onChange={handleInputChange}
                    min="1"
                    max="16"
                  />
                </div>

                <div className="form-group">
                  <label>Worker Connections</label>
                  <input
                    type="number"
                    name="worker_connections"
                    value={formData.worker_connections}
                    onChange={handleInputChange}
                    min="512"
                    max="65535"
                  />
                </div>
              </div>
            </div>

            {/* Features */}
            <div className="config-section">
              <h3>Features</h3>
              
              <div className="checkbox-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    name="ssl_enabled"
                    checked={formData.ssl_enabled}
                    onChange={handleInputChange}
                  />
                  <span>Enable SSL/TLS</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    name="gzip_enabled"
                    checked={formData.gzip_enabled}
                    onChange={handleInputChange}
                  />
                  <span>Enable Gzip Compression</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    name="health_check_enabled"
                    checked={formData.health_check_enabled}
                    onChange={handleInputChange}
                  />
                  <span>Enable Health Check</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    name="rate_limit_enabled"
                    checked={formData.rate_limit_enabled}
                    onChange={handleInputChange}
                  />
                  <span>Enable Rate Limiting</span>
                </label>

                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    name="cache_enabled"
                    checked={formData.cache_enabled}
                    onChange={handleInputChange}
                  />
                  <span>Enable Caching</span>
                </label>
              </div>

              {formData.health_check_enabled && (
                <div className="form-group">
                  <label>Health Check Path</label>
                  <input
                    type="text"
                    name="health_check_path"
                    value={formData.health_check_path}
                    onChange={handleInputChange}
                    placeholder="/health"
                  />
                </div>
              )}
            </div>
          </div>

          <div className="modal-footer">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={handleClose}
              disabled={isCreating}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={isCreating}
            >
              {isCreating ? 'Creating...' : 'Create Cluster'}
            </button>
          </div>
        </form>
        )}
      </div>
    </div>
  );
};

export default CreateNginxClusterModal;

