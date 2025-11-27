import React, { useState } from 'react';
import { Settings, Save } from 'lucide-react';
import { clusterAPI } from '../../../api';
import toast from 'react-hot-toast';
import './ClusterSettingsTab.css';

const ClusterSettingsTab = ({ clusterId, cluster, onUpdate }) => {
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState({
    replication_mode: cluster.replication_mode || 'async',
    max_connections: cluster.parameters?.max_connections || '100',
    shared_buffers: cluster.parameters?.shared_buffers || '128MB',
  });

  const handleSave = async () => {
    try {
      await clusterAPI.updateConfig(clusterId, {
        replication_mode: formData.replication_mode,
        parameters: {
          max_connections: formData.max_connections,
          shared_buffers: formData.shared_buffers,
        },
      });
      toast.success('Configuration updated successfully');
      setIsEditing(false);
      onUpdate();
    } catch (error) {
      console.error('Failed to update config:', error);
      toast.error('Failed to update configuration');
    }
  };

  return (
    <div className="cluster-settings-tab">
      <div className="settings-header">
        <h2>Cluster Settings</h2>
        {!isEditing ? (
          <button className="btn-primary" onClick={() => setIsEditing(true)}>
            <Settings size={16} />
            Edit Configuration
          </button>
        ) : (
          <div className="settings-actions">
            <button className="btn-secondary" onClick={() => setIsEditing(false)}>Cancel</button>
            <button className="btn-primary" onClick={handleSave}>
              <Save size={16} />
              Save Changes
            </button>
          </div>
        )}
      </div>

      <div className="config-section">
        <h3>Basic Configuration</h3>
        <div className="config-form">
          <div className="form-field">
            <label>Replication Mode</label>
            <select
              value={formData.replication_mode}
              onChange={(e) => setFormData({ ...formData, replication_mode: e.target.value })}
              disabled={!isEditing}
            >
              <option value="async">Asynchronous</option>
              <option value="sync">Synchronous</option>
            </select>
          </div>

          <div className="form-field">
            <label>Max Connections</label>
            <input
              type="text"
              value={formData.max_connections}
              onChange={(e) => setFormData({ ...formData, max_connections: e.target.value })}
              disabled={!isEditing}
              placeholder="e.g., 100"
            />
          </div>

          <div className="form-field">
            <label>Shared Buffers</label>
            <input
              type="text"
              value={formData.shared_buffers}
              onChange={(e) => setFormData({ ...formData, shared_buffers: e.target.value })}
              disabled={!isEditing}
              placeholder="e.g., 128MB"
            />
          </div>
        </div>
      </div>

      <div className="info-section">
        <h3>Cluster Information</h3>
        <div className="info-list">
          <div className="info-item">
            <span className="info-label">Cluster ID:</span>
            <span className="info-value">{cluster.cluster_id || cluster.id}</span>
          </div>
          <div className="info-item">
            <span className="info-label">PostgreSQL Version:</span>
            <span className="info-value">{cluster.postgres_version}</span>
          </div>
          <div className="info-item">
            <span className="info-label">Node Count:</span>
            <span className="info-value">{cluster.nodes?.length || cluster.node_count}</span>
          </div>
          <div className="info-item">
            <span className="info-label">Created At:</span>
            <span className="info-value">{new Date(cluster.created_at).toLocaleString()}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ClusterSettingsTab;
