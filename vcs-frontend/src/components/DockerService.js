import React, { useState, useEffect } from 'react';
import { dockerAPI } from '../api';

function DockerService() {
  const [services, setServices] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedService, setSelectedService] = useState(null);
  const [showDetailsModal, setShowDetailsModal] = useState(false);

  useEffect(() => {
    loadServices();
  }, []);

  const loadServices = async () => {
    setLoading(true);
    try {
      // Maintain list in state
      setLoading(false);
    } catch (error) {
      console.error('Error loading services:', error);
      setLoading(false);
    }
  };

  const handleCreateService = async (formData) => {
    try {
      const response = await dockerAPI.create(formData);
      if (response.data.success) {
        alert('Docker service created successfully!');
        setShowCreateModal(false);
        setServices([...services, response.data.data]);
      }
    } catch (error) {
      alert('Error creating service: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleViewDetails = async (serviceId) => {
    try {
      const response = await dockerAPI.getById(serviceId);
      if (response.data.success) {
        setSelectedService(response.data.data);
        setShowDetailsModal(true);
      }
    } catch (error) {
      alert('Error fetching service details: ' + error.message);
    }
  };

  const handleOperation = async (serviceId, operation) => {
    try {
      let response;
      switch (operation) {
        case 'start':
          response = await dockerAPI.start(serviceId);
          break;
        case 'stop':
          response = await dockerAPI.stop(serviceId);
          break;
        case 'restart':
          response = await dockerAPI.restart(serviceId);
          break;
        case 'delete':
          if (!window.confirm('Are you sure you want to delete this service?')) return;
          response = await dockerAPI.delete(serviceId);
          setServices(services.filter(s => s.id !== serviceId));
          break;
        default:
          return;
      }
      if (response.data.success) {
        alert(`Service ${operation} successful!`);
        loadServices();
      }
    } catch (error) {
      alert(`Error ${operation} service: ` + error.message);
    }
  };

  return (
    <div>
      <button className="create-button" onClick={() => setShowCreateModal(true)}>
        + Create Docker Service
      </button>

      {loading ? (
        <div className="loading">Loading services...</div>
      ) : services.length === 0 ? (
        <div className="empty-state">
          <h3>No Docker services yet</h3>
          <p>Create your first service to get started</p>
        </div>
      ) : (
        <div className="cards-grid">
          {services.map((service) => (
            <div key={service.id} className="card">
              <h3>{service.service_name}</h3>
              <div className="card-info">
                <strong>Status:</strong>{' '}
                <span className={`status-badge ${service.status}`}>
                  {service.status}
                </span>
              </div>
              <div className="card-info">
                <strong>Image:</strong> {service.image}
              </div>
              <div className="card-info">
                <strong>Port:</strong> {service.port}
              </div>
              {service.endpoint && (
                <div className="card-info">
                  <strong>Endpoint:</strong> {service.endpoint}
                </div>
              )}
              
              <div className="card-actions">
                <button className="btn-sm btn-info" onClick={() => handleViewDetails(service.id)}>
                  Details
                </button>
                <button className="btn-sm btn-success" onClick={() => handleOperation(service.id, 'start')}>
                  Start
                </button>
                <button className="btn-sm btn-warning" onClick={() => handleOperation(service.id, 'stop')}>
                  Stop
                </button>
                <button className="btn-sm btn-primary" onClick={() => handleOperation(service.id, 'restart')}>
                  Restart
                </button>
                <button className="btn-sm btn-danger" onClick={() => handleOperation(service.id, 'delete')}>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreateModal && (
        <CreateDockerModal
          onClose={() => setShowCreateModal(false)}
          onCreate={handleCreateService}
        />
      )}

      {showDetailsModal && selectedService && (
        <DockerDetailsModal
          service={selectedService}
          onClose={() => setShowDetailsModal(false)}
          onRefresh={loadServices}
        />
      )}
    </div>
  );
}

function CreateDockerModal({ onClose, onCreate }) {
  const [formData, setFormData] = useState({
    service_name: '',
    image: '',
    tag: 'latest',
    port: 8080,
    cpu: 256,
    memory: 268435456, // 256MB
    replicas: 1,
    env_vars: {}
  });

  const [envKey, setEnvKey] = useState('');
  const [envValue, setEnvValue] = useState('');

  const handleAddEnv = () => {
    if (envKey && envValue) {
      setFormData({
        ...formData,
        env_vars: { ...formData.env_vars, [envKey]: envValue }
      });
      setEnvKey('');
      setEnvValue('');
    }
  };

  const handleRemoveEnv = (key) => {
    const newEnvVars = { ...formData.env_vars };
    delete newEnvVars[key];
    setFormData({ ...formData, env_vars: newEnvVars });
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onCreate(formData);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Create Docker Service</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Service Name</label>
            <input
              type="text"
              value={formData.service_name}
              onChange={(e) => setFormData({...formData, service_name: e.target.value})}
              required
            />
          </div>

          <div className="form-group">
            <label>Image</label>
            <input
              type="text"
              placeholder="nginx, redis, postgres, etc."
              value={formData.image}
              onChange={(e) => setFormData({...formData, image: e.target.value})}
              required
            />
          </div>

          <div className="form-group">
            <label>Tag</label>
            <input
              type="text"
              value={formData.tag}
              onChange={(e) => setFormData({...formData, tag: e.target.value})}
              required
            />
          </div>

          <div className="form-group">
            <label>Port</label>
            <input
              type="number"
              value={formData.port}
              onChange={(e) => setFormData({...formData, port: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>CPU (millicores)</label>
            <input
              type="number"
              value={formData.cpu}
              onChange={(e) => setFormData({...formData, cpu: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Memory (bytes)</label>
            <input
              type="number"
              value={formData.memory}
              onChange={(e) => setFormData({...formData, memory: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Replicas</label>
            <input
              type="number"
              min="1"
              value={formData.replicas}
              onChange={(e) => setFormData({...formData, replicas: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Environment Variables</label>
            <div style={{ display: 'flex', gap: '5px', marginBottom: '5px' }}>
              <input
                type="text"
                placeholder="Key"
                value={envKey}
                onChange={(e) => setEnvKey(e.target.value)}
                style={{ flex: 1 }}
              />
              <input
                type="text"
                placeholder="Value"
                value={envValue}
                onChange={(e) => setEnvValue(e.target.value)}
                style={{ flex: 1 }}
              />
              <button type="button" className="btn-sm btn-primary" onClick={handleAddEnv}>
                Add
              </button>
            </div>
            {Object.keys(formData.env_vars).length > 0 && (
              <div className="list">
                {Object.entries(formData.env_vars).map(([key, value]) => (
                  <div key={key} className="list-item">
                    <span><strong>{key}</strong> = {value}</span>
                    <button 
                      type="button" 
                      className="btn-sm btn-danger" 
                      onClick={() => handleRemoveEnv(key)}
                    >
                      Remove
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="modal-actions">
            <button type="button" className="btn btn-danger" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn btn-success">
              Create Service
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function DockerDetailsModal({ service, onClose, onRefresh }) {
  const [logs, setLogs] = useState('');
  const [showLogs, setShowLogs] = useState(false);
  const [envKey, setEnvKey] = useState('');
  const [envValue, setEnvValue] = useState('');
  const [showAddEnv, setShowAddEnv] = useState(false);

  const handleLoadLogs = async () => {
    try {
      const response = await dockerAPI.getLogs(service.id);
      if (response.data.success) {
        setLogs(response.data.data.logs || 'No logs available');
        setShowLogs(true);
      }
    } catch (error) {
      alert('Error loading logs: ' + error.message);
    }
  };

  const handleUpdateEnv = async (e) => {
    e.preventDefault();
    if (!envKey || !envValue) return;
    
    try {
      const newEnvVars = { ...service.env_vars, [envKey]: envValue };
      const response = await dockerAPI.updateEnv(service.id, newEnvVars);
      if (response.data.success) {
        alert('Environment variable updated successfully!');
        setEnvKey('');
        setEnvValue('');
        setShowAddEnv(false);
        onRefresh();
      }
    } catch (error) {
      alert('Error updating environment: ' + error.message);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-large" onClick={(e) => e.stopPropagation()}>
        <h2>{service.service_name}</h2>
        
        <div className="card-info">
          <strong>Status:</strong> <span className={`status-badge ${service.status}`}>{service.status}</span>
        </div>
        <div className="card-info">
          <strong>Image:</strong> {service.image}:{service.tag}
        </div>
        <div className="card-info">
          <strong>Port:</strong> {service.port}
        </div>
        {service.endpoint && (
          <div className="card-info">
            <strong>Endpoint:</strong> {service.endpoint}
          </div>
        )}
        <div className="card-info">
          <strong>Replicas:</strong> {service.replicas}
        </div>

        <div className="section">
          <h3>Environment Variables</h3>
          <button className="btn-sm btn-primary" onClick={() => setShowAddEnv(!showAddEnv)}>
            + Add Variable
          </button>
          
          {showAddEnv && (
            <form onSubmit={handleUpdateEnv} style={{ marginTop: '10px' }}>
              <div style={{ display: 'flex', gap: '5px', marginBottom: '5px' }}>
                <input
                  type="text"
                  placeholder="Key"
                  value={envKey}
                  onChange={(e) => setEnvKey(e.target.value)}
                  required
                  style={{ flex: 1 }}
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={envValue}
                  onChange={(e) => setEnvValue(e.target.value)}
                  required
                  style={{ flex: 1 }}
                />
              </div>
              <button type="submit" className="btn-sm btn-success">Add</button>
              <button type="button" className="btn-sm btn-danger" onClick={() => setShowAddEnv(false)}>
                Cancel
              </button>
            </form>
          )}

          {service.env_vars && Object.keys(service.env_vars).length > 0 && (
            <div className="list">
              {Object.entries(service.env_vars).map(([key, value]) => (
                <div key={key} className="list-item">
                  <span><strong>{key}</strong> = {value}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="section">
          <h3>Logs</h3>
          <button className="btn-sm btn-primary" onClick={handleLoadLogs}>
            {showLogs ? 'Refresh Logs' : 'View Logs'}
          </button>
          
          {showLogs && (
            <div className="logs-container">
              <pre>{logs}</pre>
            </div>
          )}
        </div>

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}

export default DockerService;
