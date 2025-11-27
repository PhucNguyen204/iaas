import React, { useState } from 'react';
import { clusterAPI, nginxAPI, dockerAPI } from '../api';

function StackManager() {
  const [selectedTypes, setSelectedTypes] = useState([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [stacks, setStacks] = useState([]);
  const [loading, setLoading] = useState(false);

  const infraTypes = [
    { id: 'postgres', name: 'PostgreSQL Cluster', icon: '🗄️' },
    { id: 'nginx', name: 'Nginx Service', icon: '🌐' },
    { id: 'docker', name: 'Docker Service', icon: '🐳' }
  ];

  const toggleType = (typeId) => {
    if (selectedTypes.includes(typeId)) {
      setSelectedTypes(selectedTypes.filter(t => t !== typeId));
    } else {
      setSelectedTypes([...selectedTypes, typeId]);
    }
  };

  const handleCreateStack = () => {
    if (selectedTypes.length === 0) {
      alert('Please select at least one infrastructure type');
      return;
    }
    setShowCreateModal(true);
  };

  return (
    <div className="stack-manager">
      <div className="stack-header">
        <h2>Infrastructure Stack Manager</h2>
        <p>Select infrastructure types and create them as a stack</p>
      </div>

      <div className="infra-selector">
        <h3>Select Infrastructure Types:</h3>
        <div className="infra-types">
          {infraTypes.map(type => (
            <div
              key={type.id}
              className={`infra-type-card ${selectedTypes.includes(type.id) ? 'selected' : ''}`}
              onClick={() => toggleType(type.id)}
            >
              <div className="infra-icon">{type.icon}</div>
              <div className="infra-name">{type.name}</div>
              <div className="infra-checkbox">
                {selectedTypes.includes(type.id) && <span>✓</span>}
              </div>
            </div>
          ))}
        </div>
      </div>

      <button 
        className="create-stack-button" 
        onClick={handleCreateStack}
        disabled={selectedTypes.length === 0}
      >
        Create Stack ({selectedTypes.length} {selectedTypes.length === 1 ? 'service' : 'services'})
      </button>

      {loading && (
        <div className="loading-overlay">
          <div className="spinner"></div>
          <p>Creating infrastructure stack...</p>
        </div>
      )}

      <div className="stacks-section">
        <h3>Deployed Stacks</h3>
        {stacks.length === 0 ? (
          <div className="empty-state">
            <p>No stacks deployed yet. Create your first stack above!</p>
          </div>
        ) : (
          <div className="stacks-grid">
            {stacks.map(stack => (
              <StackCard key={stack.id} stack={stack} />
            ))}
          </div>
        )}
      </div>

      {showCreateModal && (
        <CreateStackModal
          selectedTypes={selectedTypes}
          onClose={() => setShowCreateModal(false)}
          onCreate={(stackData) => handleCreateStack(stackData)}
          setLoading={setLoading}
          setStacks={setStacks}
          stacks={stacks}
        />
      )}
    </div>
  );
}

function CreateStackModal({ selectedTypes, onClose, setLoading, setStacks, stacks }) {
  const [stackName, setStackName] = useState('');
  
  // PostgreSQL Cluster Config - Required fields only + common options
  const [postgresConfig, setPostgresConfig] = useState({
    cluster_name: '',
    postgres_version: '15',
    node_count: 3,
    cpu_per_node: 1000000000, // 1 CPU in nanocores
    memory_per_node: 1073741824, // 1GB in bytes
    storage_per_node: 10, // GB
    postgres_password: '',
    replication_mode: 'async'
  });

  // Nginx Config - Required fields only
  const [nginxConfig, setNginxConfig] = useState({
    name: '',
    port: 8080,
    ssl_port: 8443,
    config: `events {
  worker_connections 1024;
}
http {
  server {
    listen 8080;
    location / {
      return 200 'OK';
    }
  }
}`,
    cpu_limit: 500000000, // 0.5 CPU in nanocores
    memory_limit: 536870912 // 512MB in bytes
  });

  // Docker Service Config - Required fields
  const [dockerConfig, setDockerConfig] = useState({
    name: '',
    image: '',
    image_tag: 'latest',
    ports: [{ container_port: 8080, host_port: 0, protocol: 'tcp' }],
    env_vars: [],
    restart_policy: 'unless-stopped'
  });

  const [newEnvVar, setNewEnvVar] = useState({ key: '', value: '' });
  const [creationStatus, setCreationStatus] = useState({});

  const handleCreate = async (e) => {
    e.preventDefault();
    setLoading(true);
    
    const results = {
      postgres: null,
      nginx: null,
      docker: null
    };

    try {
      // Create PostgreSQL cluster
      if (selectedTypes.includes('postgres')) {
        setCreationStatus(prev => ({ ...prev, postgres: 'creating' }));
        try {
          console.log('Creating PostgreSQL with config:', postgresConfig);
          const response = await clusterAPI.create(postgresConfig);
          console.log('PostgreSQL response:', response);
          results.postgres = response.data.data;
          setCreationStatus(prev => ({ ...prev, postgres: 'success' }));
        } catch (error) {
          console.error('Postgres error:', error);
          console.error('Postgres error response:', error.response?.data);
          setCreationStatus(prev => ({ ...prev, postgres: 'failed' }));
        }
      }

      // Create Nginx service
      if (selectedTypes.includes('nginx')) {
        setCreationStatus(prev => ({ ...prev, nginx: 'creating' }));
        try {
          console.log('Creating Nginx with config:', nginxConfig);
          const response = await nginxAPI.create(nginxConfig);
          console.log('Nginx response:', response);
          results.nginx = response.data.data;
          setCreationStatus(prev => ({ ...prev, nginx: 'success' }));
        } catch (error) {
          console.error('Nginx error:', error);
          console.error('Nginx error response:', error.response?.data);
          setCreationStatus(prev => ({ ...prev, nginx: 'failed' }));
        }
      }

      // Create Docker service
      if (selectedTypes.includes('docker')) {
        setCreationStatus(prev => ({ ...prev, docker: 'creating' }));
        try {
          const response = await dockerAPI.create(dockerConfig);
          results.docker = response.data.data;
          setCreationStatus(prev => ({ ...prev, docker: 'success' }));
        } catch (error) {
          console.error('Docker error:', error);
          setCreationStatus(prev => ({ ...prev, docker: 'failed' }));
        }
      }

      // Add to stacks list
      const newStack = {
        id: Date.now(),
        name: stackName || 'Unnamed Stack',
        createdAt: new Date().toISOString(),
        services: results
      };
      setStacks([...stacks, newStack]);

      // Show success message
      const successCount = Object.values(results).filter(r => r !== null).length;
      alert(`Stack created! ${successCount} service(s) deployed successfully.`);
      
      onClose();
    } catch (error) {
      alert('Error creating stack: ' + error.message);
    } finally {
      setLoading(false);
      setCreationStatus({});
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-large" onClick={(e) => e.stopPropagation()}>
        <h2>Create Infrastructure Stack</h2>
        
        <form onSubmit={handleCreate}>
          <div className="form-group">
            <label>Stack Name</label>
            <input
              type="text"
              value={stackName}
              onChange={(e) => setStackName(e.target.value)}
              placeholder="Optional: My Application Stack"
            />
          </div>

          {selectedTypes.includes('postgres') && (
            <div className="config-section">
              <h3>🗄️ PostgreSQL Cluster</h3>
              {creationStatus.postgres && (
                <div className={`status-message ${creationStatus.postgres}`}>
                  {creationStatus.postgres === 'creating' && '⏳ Creating cluster...'}
                  {creationStatus.postgres === 'success' && '✅ Cluster created!'}
                  {creationStatus.postgres === 'failed' && '❌ Creation failed'}
                </div>
              )}
              
              <div className="form-row">
                <div className="form-group">
                  <label>Cluster Name *</label>
                  <input
                    type="text"
                    value={postgresConfig.cluster_name}
                    onChange={(e) => setPostgresConfig({...postgresConfig, cluster_name: e.target.value})}
                    placeholder="my-postgres-cluster"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Password * (min 8 chars)</label>
                  <input
                    type="password"
                    value={postgresConfig.postgres_password}
                    onChange={(e) => setPostgresConfig({...postgresConfig, postgres_password: e.target.value})}
                    placeholder="••••••••"
                    required
                    minLength={8}
                  />
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>Version</label>
                  <select
                    value={postgresConfig.postgres_version}
                    onChange={(e) => setPostgresConfig({...postgresConfig, postgres_version: e.target.value})}
                  >
                    <option value="16">PostgreSQL 16</option>
                    <option value="15">PostgreSQL 15</option>
                    <option value="14">PostgreSQL 14</option>
                    <option value="13">PostgreSQL 13</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Nodes (1-10)</label>
                  <input
                    type="number"
                    min="1"
                    max="10"
                    value={postgresConfig.node_count}
                    onChange={(e) => setPostgresConfig({...postgresConfig, node_count: parseInt(e.target.value)})}
                  />
                </div>
                <div className="form-group">
                  <label>Replication</label>
                  <select
                    value={postgresConfig.replication_mode}
                    onChange={(e) => setPostgresConfig({...postgresConfig, replication_mode: e.target.value})}
                  >
                    <option value="async">Async (Faster)</option>
                    <option value="sync">Sync (Safer)</option>
                  </select>
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>CPU per Node (cores)</label>
                  <select
                    value={postgresConfig.cpu_per_node}
                    onChange={(e) => setPostgresConfig({...postgresConfig, cpu_per_node: parseInt(e.target.value)})}
                  >
                    <option value="500000000">0.5 CPU</option>
                    <option value="1000000000">1 CPU</option>
                    <option value="2000000000">2 CPUs</option>
                    <option value="4000000000">4 CPUs</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Memory per Node</label>
                  <select
                    value={postgresConfig.memory_per_node}
                    onChange={(e) => setPostgresConfig({...postgresConfig, memory_per_node: parseInt(e.target.value)})}
                  >
                    <option value="536870912">512 MB</option>
                    <option value="1073741824">1 GB</option>
                    <option value="2147483648">2 GB</option>
                    <option value="4294967296">4 GB</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Storage per Node (GB)</label>
                  <input
                    type="number"
                    min="5"
                    max="1000"
                    value={postgresConfig.storage_per_node}
                    onChange={(e) => setPostgresConfig({...postgresConfig, storage_per_node: parseInt(e.target.value)})}
                  />
                </div>
              </div>
            </div>
          )}

          {selectedTypes.includes('nginx') && (
            <div className="config-section">
              <h3>🌐 Nginx Service</h3>
              {creationStatus.nginx && (
                <div className={`status-message ${creationStatus.nginx}`}>
                  {creationStatus.nginx === 'creating' && '⏳ Creating service...'}
                  {creationStatus.nginx === 'success' && '✅ Service created!'}
                  {creationStatus.nginx === 'failed' && '❌ Creation failed'}
                </div>
              )}
              
              <div className="form-row">
                <div className="form-group">
                  <label>Service Name *</label>
                  <input
                    type="text"
                    value={nginxConfig.name}
                    onChange={(e) => setNginxConfig({...nginxConfig, name: e.target.value})}
                    placeholder="my-nginx"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>HTTP Port</label>
                  <input
                    type="number"
                    min="1024"
                    max="65535"
                    value={nginxConfig.port}
                    onChange={(e) => setNginxConfig({...nginxConfig, port: parseInt(e.target.value)})}
                  />
                </div>
                <div className="form-group">
                  <label>HTTPS Port (optional)</label>
                  <input
                    type="number"
                    min="1024"
                    max="65535"
                    value={nginxConfig.ssl_port}
                    onChange={(e) => setNginxConfig({...nginxConfig, ssl_port: parseInt(e.target.value)})}
                  />
                </div>
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label>CPU Limit</label>
                  <select
                    value={nginxConfig.cpu_limit}
                    onChange={(e) => setNginxConfig({...nginxConfig, cpu_limit: parseInt(e.target.value)})}
                  >
                    <option value="250000000">0.25 CPU</option>
                    <option value="500000000">0.5 CPU</option>
                    <option value="1000000000">1 CPU</option>
                    <option value="2000000000">2 CPUs</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Memory Limit</label>
                  <select
                    value={nginxConfig.memory_limit}
                    onChange={(e) => setNginxConfig({...nginxConfig, memory_limit: parseInt(e.target.value)})}
                  >
                    <option value="268435456">256 MB</option>
                    <option value="536870912">512 MB</option>
                    <option value="1073741824">1 GB</option>
                  </select>
                </div>
              </div>

              <div className="form-group">
                <label>Nginx Config (optional - default provided)</label>
                <textarea
                  rows="4"
                  value={nginxConfig.config}
                  onChange={(e) => setNginxConfig({...nginxConfig, config: e.target.value})}
                  style={{ fontFamily: 'monospace', fontSize: '12px' }}
                />
                <small>Basic HTTP server config is pre-filled</small>
              </div>
            </div>
          )}

          {selectedTypes.includes('docker') && (
            <div className="config-section">
              <h3>🐳 Docker Service</h3>
              {creationStatus.docker && (
                <div className={`status-message ${creationStatus.docker}`}>
                  {creationStatus.docker === 'creating' && '⏳ Creating service...'}
                  {creationStatus.docker === 'success' && '✅ Service created!'}
                  {creationStatus.docker === 'failed' && '❌ Creation failed'}
                </div>
              )}
              
              <div className="form-row">
                <div className="form-group">
                  <label>Service Name *</label>
                  <input
                    type="text"
                    value={dockerConfig.name}
                    onChange={(e) => setDockerConfig({...dockerConfig, name: e.target.value})}
                    placeholder="my-app"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Docker Image *</label>
                  <input
                    type="text"
                    value={dockerConfig.image}
                    onChange={(e) => setDockerConfig({...dockerConfig, image: e.target.value})}
                    placeholder="nginx, redis, postgres..."
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Image Tag</label>
                  <input
                    type="text"
                    value={dockerConfig.image_tag}
                    onChange={(e) => setDockerConfig({...dockerConfig, image_tag: e.target.value})}
                    placeholder="latest"
                  />
                </div>
              </div>

              <div className="form-group">
                <label>Port Mapping *</label>
                <div className="form-row">
                  <input
                    type="number"
                    placeholder="Container Port"
                    value={dockerConfig.ports[0].container_port}
                    onChange={(e) => {
                      const newPorts = [...dockerConfig.ports];
                      newPorts[0].container_port = parseInt(e.target.value) || 8080;
                      setDockerConfig({...dockerConfig, ports: newPorts});
                    }}
                    style={{ flex: 1 }}
                  />
                  <span style={{ padding: '0 10px' }}>→</span>
                  <input
                    type="number"
                    placeholder="Host Port (0 = auto)"
                    value={dockerConfig.ports[0].host_port}
                    onChange={(e) => {
                      const newPorts = [...dockerConfig.ports];
                      newPorts[0].host_port = parseInt(e.target.value) || 0;
                      setDockerConfig({...dockerConfig, ports: newPorts});
                    }}
                    style={{ flex: 1 }}
                  />
                </div>
                <small>Container port → Host port (use 0 for auto-assign)</small>
              </div>

              <div className="form-group">
                <label>Environment Variables (optional)</label>
                <div className="env-vars-list">
                  {dockerConfig.env_vars.map((env, idx) => (
                    <div key={idx} className="env-var-item">
                      <span><strong>{env.key}</strong> = {env.value}</span>
                      <button
                        type="button"
                        className="btn-sm btn-danger"
                        onClick={() => {
                          const newEnvVars = dockerConfig.env_vars.filter((_, i) => i !== idx);
                          setDockerConfig({...dockerConfig, env_vars: newEnvVars});
                        }}
                      >
                        ×
                      </button>
                    </div>
                  ))}
                </div>
                <div className="form-row" style={{ marginTop: '10px' }}>
                  <input
                    type="text"
                    placeholder="KEY"
                    value={newEnvVar.key}
                    onChange={(e) => setNewEnvVar({...newEnvVar, key: e.target.value})}
                    style={{ flex: 1 }}
                  />
                  <input
                    type="text"
                    placeholder="value"
                    value={newEnvVar.value}
                    onChange={(e) => setNewEnvVar({...newEnvVar, value: e.target.value})}
                    style={{ flex: 1 }}
                  />
                  <button
                    type="button"
                    className="btn-sm btn-primary"
                    onClick={() => {
                      if (newEnvVar.key && newEnvVar.value) {
                        setDockerConfig({
                          ...dockerConfig,
                          env_vars: [...dockerConfig.env_vars, newEnvVar]
                        });
                        setNewEnvVar({ key: '', value: '' });
                      }
                    }}
                  >
                    Add
                  </button>
                </div>
              </div>

              <div className="form-group">
                <label>Restart Policy</label>
                <select
                  value={dockerConfig.restart_policy}
                  onChange={(e) => setDockerConfig({...dockerConfig, restart_policy: e.target.value})}
                >
                  <option value="no">No</option>
                  <option value="always">Always</option>
                  <option value="unless-stopped">Unless Stopped</option>
                  <option value="on-failure">On Failure</option>
                </select>
              </div>
            </div>
          )}

          <div className="modal-actions">
            <button type="button" className="btn btn-danger" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn btn-success">
              Create Stack
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function StackCard({ stack }) {
  return (
    <div className="stack-card">
      <h3>{stack.name}</h3>
      <div className="stack-info">
        <small>Created: {new Date(stack.createdAt).toLocaleString()}</small>
      </div>
      <div className="stack-services">
        {stack.services.postgres && (
          <div className="service-badge postgres">🗄️ PostgreSQL</div>
        )}
        {stack.services.nginx && (
          <div className="service-badge nginx">🌐 Nginx</div>
        )}
        {stack.services.docker && (
          <div className="service-badge docker">🐳 Docker</div>
        )}
      </div>
    </div>
  );
}

export default StackManager;
