import React, { useState } from 'react';
import { Save, RefreshCw, Pause, Play, Trash2, Copy } from 'lucide-react';
import toast from 'react-hot-toast';
import './StackConfigTab.css';

const StackConfigTab = ({ stack, onRefresh }) => {
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState({
    name: stack.name,
    description: stack.description || '',
    environment: stack.environment,
    tags: stack.tags || {}
  });

  const handleSave = async () => {
    try {
      // await stackAPI.update(stack.id, formData);
      toast.success('Stack configuration updated');
      setEditing(false);
      onRefresh();
    } catch (error) {
      toast.error('Failed to update configuration');
    }
  };

  const handleCancel = () => {
    setFormData({
      name: stack.name,
      description: stack.description || '',
      environment: stack.environment,
      tags: stack.tags || {}
    });
    setEditing(false);
  };

  return (
    <div className="stack-config-tab">
      {/* Basic Information */}
      <div className="config-section">
        <div className="section-header">
          <h4>Basic Information</h4>
          {!editing ? (
            <button className="btn btn-secondary" onClick={() => setEditing(true)}>
              Edit
            </button>
          ) : (
            <div className="edit-buttons">
              <button className="btn btn-primary" onClick={handleSave}>
                <Save size={16} />
                Save Changes
              </button>
              <button className="btn btn-secondary" onClick={handleCancel}>
                Cancel
              </button>
            </div>
          )}
        </div>

        <div className="config-form">
          <div className="form-group">
            <label>Stack Name</label>
            {editing ? (
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            ) : (
              <div className="form-value">{stack.name}</div>
            )}
          </div>

          <div className="form-group">
            <label>Description</label>
            {editing ? (
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
              />
            ) : (
              <div className="form-value">{stack.description || 'No description'}</div>
            )}
          </div>

          <div className="form-group">
            <label>Environment</label>
            {editing ? (
              <select
                value={formData.environment}
                onChange={(e) => setFormData({ ...formData, environment: e.target.value })}
              >
                <option value="development">Development</option>
                <option value="staging">Staging</option>
                <option value="production">Production</option>
              </select>
            ) : (
              <div className="form-value">{stack.environment}</div>
            )}
          </div>

          <div className="form-group">
            <label>Stack ID</label>
            <div className="form-value code">{stack.id}</div>
          </div>

          <div className="form-group">
            <label>Tags</label>
            {editing ? (
              <input
                type="text"
                placeholder="Comma-separated tags"
                value={Object.values(formData.tags).join(', ')}
                onChange={(e) => {
                  const tags = e.target.value.split(',').map(t => t.trim());
                  setFormData({ ...formData, tags: Object.fromEntries(tags.map((t, i) => [i, t])) });
                }}
              />
            ) : (
              <div className="tags-display">
                {Object.values(stack.tags || {}).map((tag, idx) => (
                  <span key={idx} className="tag">#{tag}</span>
                ))}
                {Object.keys(stack.tags || {}).length === 0 && (
                  <span className="no-tags">No tags</span>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Resource Dependencies */}
      <div className="config-section">
        <h4>Resource Dependencies</h4>
        <div className="dependencies-order">
          <h5>Creation Order:</h5>
          <ol className="order-list">
            {stack.resources
              ?.sort((a, b) => (a.order || 0) - (b.order || 0))
              .map((resource, idx) => (
                <li key={idx}>
                  <span className="resource-name">{resource.resource_name || resource.resource_type}</span>
                  <span className="resource-role">({resource.role})</span>
                  {resource.depends_on && resource.depends_on.length > 0 && (
                    <span className="depends-on">
                      depends on: {resource.depends_on.join(', ')}
                    </span>
                  )}
                </li>
              ))}
          </ol>
        </div>
      </div>

      {/* Stack Operations */}
      <div className="config-section">
        <h4>Stack Operations</h4>
        <div className="operations-grid">
          <button className="operation-btn">
            <RefreshCw size={20} />
            <span>Restart All Services</span>
          </button>
          <button className="operation-btn">
            <Pause size={20} />
            <span>Stop All Services</span>
          </button>
          <button className="operation-btn">
            <Play size={20} />
            <span>Start All Services</span>
          </button>
          <button className="operation-btn">
            <Copy size={20} />
            <span>Clone Stack</span>
          </button>
          <button className="operation-btn">
            <Save size={20} />
            <span>Export Template</span>
          </button>
          <button className="operation-btn danger">
            <Trash2 size={20} />
            <span>Delete Stack</span>
          </button>
        </div>
      </div>

      {/* Metadata */}
      <div className="config-section">
        <h4>Metadata</h4>
        <div className="metadata-grid">
          <div className="metadata-item">
            <span className="label">Created At:</span>
            <span className="value">{new Date(stack.created_at).toLocaleString()}</span>
          </div>
          <div className="metadata-item">
            <span className="label">Updated At:</span>
            <span className="value">{new Date(stack.updated_at).toLocaleString()}</span>
          </div>
          <div className="metadata-item">
            <span className="label">Total Resources:</span>
            <span className="value">{stack.resources?.length || 0}</span>
          </div>
          <div className="metadata-item">
            <span className="label">Status:</span>
            <span className="value">{stack.status}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StackConfigTab;
