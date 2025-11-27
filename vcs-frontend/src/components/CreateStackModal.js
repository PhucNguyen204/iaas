import React, { useState } from 'react';
import { X, Plus, Trash2, Database, Globe, Container, Server, Shield, Network } from 'lucide-react';
import { stackAPI } from '../api';
import toast from 'react-hot-toast';
import './CreateStackModal.css';

const CreateStackModal = ({ isOpen, onClose, onSuccess }) => {
  const [formData, setFormData] = useState({
    name: '',
    environment: 'development',
    description: '',
    resources: []
  });

  // Default spec for POSTGRES_CLUSTER
  const getInitialPostgresSpec = () => ({
    postgres_version: '17',
    node_count: 2,
    replication_mode: 'async',
    postgres_password: ''
  });

  const [newResource, setNewResource] = useState({
    type: 'POSTGRES_CLUSTER',
    name: '',
    role: 'database',
    spec: getInitialPostgresSpec()
  });

  const [isCreating, setIsCreating] = useState(false);

  if (!isOpen) return null;

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleResourceChange = (e) => {
    const { name, value } = e.target;
    setNewResource(prev => ({
      ...prev,
      [name]: value,
      // Reset spec when type changes
      ...(name === 'type' ? { spec: getDefaultSpec(value) } : {})
    }));
  };

  const getDefaultSpec = (type) => {
    switch (type) {
      case 'POSTGRES_CLUSTER':
        return {
          postgres_version: '17',
          node_count: 2,
          replication_mode: 'async',
          postgres_password: ''
        };
      case 'NGINX_GATEWAY':
        return {
          port: 8080,
          config: `server {
    listen 80;
    server_name localhost;
    
    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}`
        };
      case 'DOCKER_SERVICE':
        return {
          image: '',
          image_tag: 'latest',
          ports: [{ container_port: 80, host_port: 0, protocol: 'tcp' }],
          env_vars: []
        };
      default:
        return {};
    }
  };

  const handleConfigChange = (key, value) => {
    setNewResource(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        [key]: value
      }
    }));
  };

  const handleArrayConfigChange = (arrayKey, index, key, value) => {
    setNewResource(prev => {
      const newArray = [...(prev.spec[arrayKey] || [])];
      newArray[index] = { ...newArray[index], [key]: value };
      return {
        ...prev,
        spec: {
          ...prev.spec,
          [arrayKey]: newArray
        }
      };
    });
  };

  const addArrayItem = (arrayKey, defaultItem) => {
    setNewResource(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        [arrayKey]: [...(prev.spec[arrayKey] || []), defaultItem]
      }
    }));
  };

  const removeArrayItem = (arrayKey, index) => {
    setNewResource(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        [arrayKey]: prev.spec[arrayKey].filter((_, i) => i !== index)
      }
    }));
  };

  const validateResource = () => {
    if (!newResource.name) {
      toast.error('Please enter resource name');
      return false;
    }

    if (newResource.type === 'POSTGRES_CLUSTER') {
      if (!newResource.spec.postgres_password || newResource.spec.postgres_password.length < 8) {
        toast.error('Postgres password is required (min 8 characters)');
        return false;
      }
    }

    if (newResource.type === 'NGINX_GATEWAY') {
      if (!newResource.spec.config) {
        toast.error('Nginx config is required');
        return false;
      }
    }

    if (newResource.type === 'DOCKER_SERVICE') {
      if (!newResource.spec.image) {
        toast.error('Docker image is required');
        return false;
      }
      if (!newResource.spec.image_tag) {
        toast.error('Docker image tag is required');
        return false;
      }
    }

    return true;
  };

  const addResource = () => {
    if (!validateResource()) return;

    const resourceWithOrder = {
      resource_type: newResource.type,
      resource_name: newResource.name,
      role: newResource.role,
      spec: newResource.spec,
      order: formData.resources.length + 1
    };

    setFormData(prev => ({
      ...prev,
      resources: [...prev.resources, resourceWithOrder]
    }));

    // Reset to new resource with defaults
    setNewResource({
      type: 'POSTGRES_CLUSTER',
      name: '',
      role: 'database',
      spec: getInitialPostgresSpec()
    });

    toast.success('Resource added to stack');
  };

  const removeResource = (index) => {
    setFormData(prev => ({
      ...prev,
      resources: prev.resources.filter((_, i) => i !== index)
    }));
    toast.success('Resource removed');
  };

  const handleSubmit = async () => {
    if (!formData.name) {
      toast.error('Please enter stack name');
      return;
    }

    if (formData.resources.length === 0) {
      toast.error('Please add at least one resource');
      return;
    }

    try {
      setIsCreating(true);
      const loadingToast = toast.loading('Creating stack and provisioning infrastructure...');

      console.log('Creating stack with data:', JSON.stringify(formData, null, 2));
      const response = await stackAPI.create(formData);
      console.log('Stack created:', response.data);

      toast.dismiss(loadingToast);
      toast.success(
        `Stack "${formData.name}" is being created! Infrastructure provisioning in progress...`,
        { duration: 5000 }
      );

      onSuccess();
      handleClose();
    } catch (error) {
      console.error('Error creating stack:', error);
      console.error('Error response:', error.response?.data);
      toast.error(error.response?.data?.error || 'Failed to create stack');
    } finally {
      setIsCreating(false);
    }
  };

  const handleClose = () => {
    setFormData({
      name: '',
      environment: 'development',
      description: '',
      resources: []
    });
    setNewResource({
      type: 'POSTGRES_CLUSTER',
      name: '',
      role: 'database',
      spec: getInitialPostgresSpec()
    });
    onClose();
  };

  const getResourceIcon = (type) => {
    switch (type) {
      case 'POSTGRES_CLUSTER':
        return <Database size={18} />;
      case 'NGINX_GATEWAY':
        return <Globe size={18} />;
      case 'DOCKER_SERVICE':
        return <Container size={18} />;
      default:
        return <Server size={18} />;
    }
  };

  const getResourceColor = (type) => {
    switch (type) {
      case 'POSTGRES_CLUSTER':
        return '#336791';
      case 'NGINX_GATEWAY':
        return '#009639';
      case 'DOCKER_SERVICE':
        return '#2496ED';
      default:
        return '#6b7280';
    }
  };

  const renderPostgresConfig = () => {
    const { spec } = newResource;
    return (
      <div className="config-container">
        <div className="config-section">
          <div className="config-section-header">
            <div className="config-section-title">
              <Database size={16} />
              <span>PostgreSQL Configuration</span>
            </div>
          </div>
          <div className="config-section-content">
            <div className="config-grid">
              <div className="form-group">
                <label><Database size={14} /> PostgreSQL Version</label>
                <select
                  value={spec.postgres_version || '17'}
                  onChange={(e) => handleConfigChange('postgres_version', e.target.value)}
                >
                  <option value="15">PostgreSQL 15</option>
                  <option value="16">PostgreSQL 16</option>
                  <option value="17">PostgreSQL 17</option>
                </select>
              </div>
              <div className="form-group">
                <label><Shield size={14} /> Password *</label>
                <input
                  type="password"
                  value={spec.postgres_password || ''}
                  onChange={(e) => handleConfigChange('postgres_password', e.target.value)}
                  placeholder="Min 8 characters"
                />
              </div>
              <div className="form-group">
                <label><Server size={14} /> Node Count</label>
                <input
                  type="number"
                  min="1"
                  max="10"
                  value={spec.node_count || 2}
                  onChange={(e) => handleConfigChange('node_count', parseInt(e.target.value))}
                />
              </div>
              <div className="form-group">
                <label><Network size={14} /> Replication Mode</label>
                <select
                  value={spec.replication_mode || 'async'}
                  onChange={(e) => handleConfigChange('replication_mode', e.target.value)}
                >
                  <option value="async">Asynchronous</option>
                  <option value="sync">Synchronous</option>
                </select>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const renderNginxConfig = () => {
    const { spec } = newResource;
    return (
      <div className="config-container">
        <div className="config-section">
          <div className="config-section-header">
            <div className="config-section-title">
              <Globe size={16} />
              <span>Nginx Configuration</span>
            </div>
          </div>
          <div className="config-section-content">
            <div className="config-grid">
              <div className="form-group">
                <label>HTTP Port *</label>
                <input
                  type="number"
                  min="1"
                  max="65535"
                  value={spec.port || 8080}
                  onChange={(e) => handleConfigChange('port', parseInt(e.target.value))}
                />
              </div>
            </div>
            <div className="form-group full-width">
              <label>Nginx Configuration *</label>
              <textarea
                value={spec.config || ''}
                onChange={(e) => handleConfigChange('config', e.target.value)}
                placeholder="Enter nginx configuration..."
                rows="8"
                className="code-textarea"
              />
            </div>
          </div>
        </div>
      </div>
    );
  };

  const renderDockerConfig = () => {
    const { spec } = newResource;
    return (
      <div className="config-container">
        <div className="config-section">
          <div className="config-section-header">
            <div className="config-section-title">
              <Container size={16} />
              <span>Docker Configuration</span>
            </div>
          </div>
          <div className="config-section-content">
            <div className="config-grid">
              <div className="form-group">
                <label>Image Name *</label>
                <input
                  type="text"
                  value={spec.image || ''}
                  onChange={(e) => handleConfigChange('image', e.target.value)}
                  placeholder="e.g., nginx, redis, mongo"
                />
              </div>
              <div className="form-group">
                <label>Image Tag *</label>
                <input
                  type="text"
                  value={spec.image_tag || ''}
                  onChange={(e) => handleConfigChange('image_tag', e.target.value)}
                  placeholder="e.g., latest, alpine"
                />
              </div>
            </div>
            
            {/* Port Mappings */}
            <div className="form-group full-width">
              <label>Port Mappings</label>
              {(spec.ports || []).map((port, index) => (
                <div key={index} className="array-item">
                  <div className="config-grid-3">
                    <div className="form-group">
                      <label>Container Port</label>
                      <input
                        type="number"
                        min="1"
                        max="65535"
                        value={port.container_port || ''}
                        onChange={(e) => handleArrayConfigChange('ports', index, 'container_port', parseInt(e.target.value))}
                      />
                    </div>
                    <div className="form-group">
                      <label>Host Port</label>
                      <input
                        type="number"
                        min="0"
                        max="65535"
                        value={port.host_port || ''}
                        onChange={(e) => handleArrayConfigChange('ports', index, 'host_port', parseInt(e.target.value))}
                        placeholder="Auto"
                      />
                    </div>
                    <div className="form-group">
                      <label>Protocol</label>
                      <select
                        value={port.protocol || 'tcp'}
                        onChange={(e) => handleArrayConfigChange('ports', index, 'protocol', e.target.value)}
                      >
                        <option value="tcp">TCP</option>
                        <option value="udp">UDP</option>
                      </select>
                    </div>
                  </div>
                  <button
                    type="button"
                    className="btn-icon btn-danger"
                    onClick={() => removeArrayItem('ports', index)}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="btn btn-sm btn-secondary"
                onClick={() => addArrayItem('ports', { container_port: 80, host_port: 0, protocol: 'tcp' })}
              >
                <Plus size={14} /> Add Port
              </button>
            </div>

            {/* Environment Variables */}
            <div className="form-group full-width">
              <label>Environment Variables</label>
              {(spec.env_vars || []).map((env, index) => (
                <div key={index} className="array-item">
                  <div className="config-grid-2">
                    <div className="form-group">
                      <label>Key</label>
                      <input
                        type="text"
                        value={env.key || ''}
                        onChange={(e) => handleArrayConfigChange('env_vars', index, 'key', e.target.value)}
                        placeholder="VARIABLE_NAME"
                      />
                    </div>
                    <div className="form-group">
                      <label>Value</label>
                      <input
                        type="text"
                        value={env.value || ''}
                        onChange={(e) => handleArrayConfigChange('env_vars', index, 'value', e.target.value)}
                        placeholder="value"
                      />
                    </div>
                  </div>
                  <button
                    type="button"
                    className="btn-icon btn-danger"
                    onClick={() => removeArrayItem('env_vars', index)}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="btn btn-sm btn-secondary"
                onClick={() => addArrayItem('env_vars', { key: '', value: '' })}
              >
                <Plus size={14} /> Add Variable
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const renderConfigFields = () => {
    switch (newResource.type) {
      case 'POSTGRES_CLUSTER':
        return renderPostgresConfig();
      case 'NGINX_GATEWAY':
        return renderNginxConfig();
      case 'DOCKER_SERVICE':
        return renderDockerConfig();
      default:
        return null;
    }
  };

  const getResourceSummary = (resource) => {
    switch (resource.resource_type) {
      case 'POSTGRES_CLUSTER':
        return `${resource.spec.node_count || 3} nodes • v${resource.spec.postgres_version || '17'} • ${resource.spec.replication_mode || 'async'}`;
      case 'NGINX_GATEWAY':
        return `Port ${resource.spec.port || 8080}${resource.spec.ssl_port ? ` / SSL ${resource.spec.ssl_port}` : ''}`;
      case 'DOCKER_SERVICE':
        return `${resource.spec.image || 'unknown'}:${resource.spec.image_tag || 'latest'}`;
      default:
        return '';
    }
  };

  return (
    <div className="modal-overlay" onClick={handleClose}>
      <div className="modal-content create-stack-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Create New Stack</h2>
          <button className="close-btn" onClick={handleClose}>
            <X size={20} />
          </button>
        </div>

        <div className="modal-body">
          {/* Basic Info */}
          <div className="form-section">
            <h3>Stack Information</h3>
            <div className="config-grid">
              <div className="form-group">
                <label>Stack Name *</label>
                <input
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleInputChange}
                  placeholder="e.g., my-production-app"
                />
              </div>
              <div className="form-group">
                <label>Environment</label>
                <select
                  name="environment"
                  value={formData.environment}
                  onChange={handleInputChange}
                >
                  <option value="development">Development</option>
                  <option value="staging">Staging</option>
                  <option value="production">Production</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea
                name="description"
                value={formData.description}
                onChange={handleInputChange}
                placeholder="Describe your stack..."
                rows="2"
              />
            </div>
          </div>

          {/* Added Resources */}
          {formData.resources.length > 0 && (
            <div className="form-section">
              <h3>Resources ({formData.resources.length})</h3>
              <div className="resources-list">
                {formData.resources.map((resource, index) => (
                  <div key={index} className="resource-item">
                    <div
                      className="resource-icon"
                      style={{ background: getResourceColor(resource.resource_type) }}
                    >
                      {getResourceIcon(resource.resource_type)}
                    </div>
                    <div className="resource-info">
                      <div className="resource-name">{resource.resource_name}</div>
                      <div className="resource-type">{getResourceSummary(resource)}</div>
                    </div>
                    <span className="resource-badge">{resource.resource_type.replace(/_/g, ' ')}</span>
                    <button
                      type="button"
                      className="btn-icon btn-danger"
                      onClick={() => removeResource(index)}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Add Resource Form */}
          <div className="form-section">
            <h3>Add Resource</h3>
            <div className="resource-form">
              <div className="config-grid">
                <div className="form-group">
                  <label>Resource Type</label>
                  <select
                    name="type"
                    value={newResource.type}
                    onChange={handleResourceChange}
                  >
                    <option value="POSTGRES_CLUSTER">PostgreSQL Cluster</option>
                    <option value="NGINX_GATEWAY">Nginx Gateway</option>
                    <option value="DOCKER_SERVICE">Docker Service</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Resource Name *</label>
                  <input
                    type="text"
                    name="name"
                    value={newResource.name}
                    onChange={handleResourceChange}
                    placeholder="e.g., main-database"
                  />
                </div>
                <div className="form-group">
                  <label>Role</label>
                  <select
                    name="role"
                    value={newResource.role}
                    onChange={handleResourceChange}
                  >
                    <option value="database">Database</option>
                    <option value="gateway">Gateway</option>
                    <option value="app">Application</option>
                    <option value="cache">Cache</option>
                    <option value="queue">Message Queue</option>
                  </select>
                </div>
              </div>

              {renderConfigFields()}

              <div className="form-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={addResource}
                >
                  <Plus size={16} />
                  Add Resource to Stack
                </button>
              </div>
            </div>
          </div>
        </div>

        <div className="modal-footer">
          <button className="btn btn-secondary" onClick={handleClose} disabled={isCreating}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={isCreating || formData.resources.length === 0}>
            {isCreating ? (
              <>
                <span className="spinner-small"></span>
                Creating Stack...
              </>
            ) : (
              <>Create Stack ({formData.resources.length} resources)</>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

export default CreateStackModal;
