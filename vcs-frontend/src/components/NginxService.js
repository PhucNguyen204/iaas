import React, { useState, useEffect } from 'react';
import { nginxAPI } from '../api';

function NginxService() {
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
      const response = await nginxAPI.create(formData);
      if (response.data.success) {
        alert('Nginx service created successfully!');
        setShowCreateModal(false);
        setServices([...services, response.data.data]);
      }
    } catch (error) {
      alert('Error creating service: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleViewDetails = async (instanceId) => {
    try {
      const response = await nginxAPI.getById(instanceId);
      if (response.data.success) {
        setSelectedService(response.data.data);
        setShowDetailsModal(true);
      }
    } catch (error) {
      alert('Error fetching service details: ' + error.message);
    }
  };

  const handleOperation = async (instanceId, operation) => {
    try {
      let response;
      switch (operation) {
        case 'start':
          response = await nginxAPI.start(instanceId);
          break;
        case 'stop':
          response = await nginxAPI.stop(instanceId);
          break;
        case 'restart':
          response = await nginxAPI.restart(instanceId);
          break;
        case 'delete':
          if (!window.confirm('Are you sure you want to delete this service?')) return;
          response = await nginxAPI.delete(instanceId);
          setServices(services.filter(s => s.id !== instanceId));
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
        + Create Nginx Service
      </button>

      {loading ? (
        <div className="loading">Loading services...</div>
      ) : services.length === 0 ? (
        <div className="empty-state">
          <h3>No Nginx services yet</h3>
          <p>Create your first service to get started</p>
        </div>
      ) : (
        <div className="cards-grid">
          {services.map((service) => (
            <div key={service.id} className="card">
              <h3>{service.instance_name}</h3>
              <div className="card-info">
                <strong>Status:</strong>{' '}
                <span className={`status-badge ${service.status}`}>
                  {service.status}
                </span>
              </div>
              <div className="card-info">
                <strong>Version:</strong> Nginx {service.version}
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
        <CreateNginxModal
          onClose={() => setShowCreateModal(false)}
          onCreate={handleCreateService}
        />
      )}

      {showDetailsModal && selectedService && (
        <NginxDetailsModal
          service={selectedService}
          onClose={() => setShowDetailsModal(false)}
          onRefresh={loadServices}
        />
      )}
    </div>
  );
}

function CreateNginxModal({ onClose, onCreate }) {
  const [formData, setFormData] = useState({
    instance_name: '',
    nginx_version: '1.24',
    port: 80,
    cpu: 256,
    memory: 268435456, // 256MB
    worker_processes: 2,
    worker_connections: 1024,
    keepalive_timeout: 65
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onCreate(formData);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Create Nginx Service</h2>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Instance Name</label>
            <input
              type="text"
              value={formData.instance_name}
              onChange={(e) => setFormData({...formData, instance_name: e.target.value})}
              required
            />
          </div>

          <div className="form-group">
            <label>Nginx Version</label>
            <select
              value={formData.nginx_version}
              onChange={(e) => setFormData({...formData, nginx_version: e.target.value})}
            >
              <option value="1.24">1.24</option>
              <option value="1.23">1.23</option>
              <option value="1.22">1.22</option>
            </select>
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
            <label>Worker Processes</label>
            <input
              type="number"
              value={formData.worker_processes}
              onChange={(e) => setFormData({...formData, worker_processes: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Worker Connections</label>
            <input
              type="number"
              value={formData.worker_connections}
              onChange={(e) => setFormData({...formData, worker_connections: parseInt(e.target.value)})}
              required
            />
          </div>

          <div className="form-group">
            <label>Keepalive Timeout (seconds)</label>
            <input
              type="number"
              value={formData.keepalive_timeout}
              onChange={(e) => setFormData({...formData, keepalive_timeout: parseInt(e.target.value)})}
              required
            />
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

function NginxDetailsModal({ service, onClose, onRefresh }) {
  const [domainName, setDomainName] = useState('');
  const [routeData, setRouteData] = useState({ path: '', backend_url: '' });
  const [showAddDomain, setShowAddDomain] = useState(false);
  const [showAddRoute, setShowAddRoute] = useState(false);

  const handleAddDomain = async (e) => {
    e.preventDefault();
    try {
      const response = await nginxAPI.addDomain(service.id, domainName);
      if (response.data.success) {
        alert('Domain added successfully!');
        setDomainName('');
        setShowAddDomain(false);
        onRefresh();
      }
    } catch (error) {
      alert('Error adding domain: ' + error.message);
    }
  };

  const handleDeleteDomain = async (domain) => {
    if (!window.confirm(`Delete domain ${domain}?`)) return;
    try {
      const response = await nginxAPI.deleteDomain(service.id, domain);
      if (response.data.success) {
        alert('Domain deleted successfully!');
        onRefresh();
      }
    } catch (error) {
      alert('Error deleting domain: ' + error.message);
    }
  };

  const handleAddRoute = async (e) => {
    e.preventDefault();
    try {
      const response = await nginxAPI.addRoute(service.id, routeData);
      if (response.data.success) {
        alert('Route added successfully!');
        setRouteData({ path: '', backend_url: '' });
        setShowAddRoute(false);
        onRefresh();
      }
    } catch (error) {
      alert('Error adding route: ' + error.message);
    }
  };

  const handleDeleteRoute = async (path) => {
    if (!window.confirm(`Delete route ${path}?`)) return;
    try {
      const response = await nginxAPI.deleteRoute(service.id, path);
      if (response.data.success) {
        alert('Route deleted successfully!');
        onRefresh();
      }
    } catch (error) {
      alert('Error deleting route: ' + error.message);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-large" onClick={(e) => e.stopPropagation()}>
        <h2>{service.instance_name}</h2>
        
        <div className="card-info">
          <strong>Status:</strong> <span className={`status-badge ${service.status}`}>{service.status}</span>
        </div>
        <div className="card-info">
          <strong>Version:</strong> Nginx {service.version}
        </div>
        <div className="card-info">
          <strong>Port:</strong> {service.port}
        </div>
        {service.endpoint && (
          <div className="card-info">
            <strong>Endpoint:</strong> {service.endpoint}
          </div>
        )}

        <div className="section">
          <h3>Domains</h3>
          <button className="btn-sm btn-primary" onClick={() => setShowAddDomain(!showAddDomain)}>
            + Add Domain
          </button>
          
          {showAddDomain && (
            <form onSubmit={handleAddDomain} style={{ marginTop: '10px' }}>
              <div className="form-group">
                <input
                  type="text"
                  placeholder="example.com"
                  value={domainName}
                  onChange={(e) => setDomainName(e.target.value)}
                  required
                />
              </div>
              <button type="submit" className="btn-sm btn-success">Add</button>
              <button type="button" className="btn-sm btn-danger" onClick={() => setShowAddDomain(false)}>
                Cancel
              </button>
            </form>
          )}

          {service.domains && service.domains.length > 0 && (
            <div className="list">
              {service.domains.map((domain, idx) => (
                <div key={idx} className="list-item">
                  <span>{domain}</span>
                  <button className="btn-sm btn-danger" onClick={() => handleDeleteDomain(domain)}>
                    Delete
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="section">
          <h3>Routes</h3>
          <button className="btn-sm btn-primary" onClick={() => setShowAddRoute(!showAddRoute)}>
            + Add Route
          </button>
          
          {showAddRoute && (
            <form onSubmit={handleAddRoute} style={{ marginTop: '10px' }}>
              <div className="form-group">
                <input
                  type="text"
                  placeholder="Path (e.g., /api)"
                  value={routeData.path}
                  onChange={(e) => setRouteData({...routeData, path: e.target.value})}
                  required
                />
              </div>
              <div className="form-group">
                <input
                  type="text"
                  placeholder="Backend URL (e.g., http://backend:8080)"
                  value={routeData.backend_url}
                  onChange={(e) => setRouteData({...routeData, backend_url: e.target.value})}
                  required
                />
              </div>
              <button type="submit" className="btn-sm btn-success">Add</button>
              <button type="button" className="btn-sm btn-danger" onClick={() => setShowAddRoute(false)}>
                Cancel
              </button>
            </form>
          )}

          {service.routes && service.routes.length > 0 && (
            <div className="list">
              {service.routes.map((route, idx) => (
                <div key={idx} className="list-item">
                  <div>
                    <strong>{route.path}</strong> → {route.backend_url}
                  </div>
                  <button className="btn-sm btn-danger" onClick={() => handleDeleteRoute(route.path)}>
                    Delete
                  </button>
                </div>
              ))}
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

export default NginxService;
