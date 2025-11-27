import React, { useState } from 'react';
import { X, ChevronRight, ChevronLeft, Check, Database, Server, Shield, Package } from 'lucide-react';
import { clusterAPI } from '../../api';
import toast from 'react-hot-toast';
import './CreatePostgreSQLClusterWizard.css';

const STEPS = [
  { id: 1, title: 'Basic Info', icon: Database },
  { id: 2, title: 'Resources', icon: Server },
  { id: 3, title: 'Advanced Config', icon: Shield },
  { id: 4, title: 'Review', icon: Package },
];

const CreatePostgreSQLClusterWizard = ({ onClose, onSuccess }) => {
  const [currentStep, setCurrentStep] = useState(1);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    // Step 1: Basic Info
    cluster_name: '',
    postgres_version: '17',
    node_count: 3,
    replication_mode: 'async',
    postgres_password: '',
    namespace: 'production',
    
    // Step 2: Resources
    cpu_per_node: 2,
    memory_per_node: 2048,
    storage_per_node: 20,
    
    // Step 3: Advanced Config
    enable_backup: true,
    backup_retention: 7,
    backup_schedule: '0 2 * * *',
    enable_haproxy: true,
    haproxy_port: 5000,
    haproxy_read_port: 5001,
    max_replication_lag: 10485760, // 10MB in bytes
    ttl: 30,
    loop_wait: 10,
    use_pg_rewind: true,
    use_slots: true,
  });

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : (type === 'number' ? parseInt(value) || 0 : value)
    }));
  };

  const validateStep = (step) => {
    switch (step) {
      case 1:
        if (!formData.cluster_name.trim()) {
          toast.error('Cluster name is required');
          return false;
        }
        if (formData.postgres_password.length < 8) {
          toast.error('Password must be at least 8 characters');
          return false;
        }
        if (formData.node_count < 1 || formData.node_count > 10) {
          toast.error('Node count must be between 1 and 10');
          return false;
        }
        return true;
      case 2:
        if (formData.cpu_per_node < 1) {
          toast.error('CPU per node must be at least 1');
          return false;
        }
        if (formData.memory_per_node < 512) {
          toast.error('Memory per node must be at least 512 MB');
          return false;
        }
        if (formData.storage_per_node < 5) {
          toast.error('Storage per node must be at least 5 GB');
          return false;
        }
        return true;
      case 3:
        return true;
      default:
        return true;
    }
  };

  const handleNext = () => {
    if (validateStep(currentStep)) {
      setCurrentStep(prev => Math.min(prev + 1, STEPS.length));
    }
  };

  const handleBack = () => {
    setCurrentStep(prev => Math.max(prev - 1, 1));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
      const response = await clusterAPI.create(formData);
      toast.success('PostgreSQL cluster created successfully!');
      if (onSuccess) {
        onSuccess(response.data.data.cluster);
      }
      onClose();
    } catch (error) {
      console.error('Failed to create cluster:', error);
      toast.error(error.response?.data?.message || 'Failed to create cluster');
    } finally {
      setIsSubmitting(false);
    }
  };

  const renderStepContent = () => {
    switch (currentStep) {
      case 1:
        return (
          <div className="wizard-form">
            <div className="form-group">
              <label>Cluster Name *</label>
              <input
                type="text"
                name="cluster_name"
                value={formData.cluster_name}
                onChange={handleInputChange}
                placeholder="e.g., production-cluster"
                required
              />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>PostgreSQL Version *</label>
                <select
                  name="postgres_version"
                  value={formData.postgres_version}
                  onChange={handleInputChange}
                >
                  <option value="17">PostgreSQL 17</option>
                  <option value="16">PostgreSQL 16</option>
                  <option value="15">PostgreSQL 15</option>
                  <option value="14">PostgreSQL 14</option>
                </select>
              </div>

              <div className="form-group">
                <label>Number of Nodes *</label>
                <input
                  type="number"
                  name="node_count"
                  value={formData.node_count}
                  onChange={handleInputChange}
                  min="1"
                  max="10"
                  required
                />
                <small>1 primary + {formData.node_count - 1} replicas</small>
              </div>
            </div>

            <div className="form-group">
              <label>Postgres Password *</label>
              <input
                type="password"
                name="postgres_password"
                value={formData.postgres_password}
                onChange={handleInputChange}
                placeholder="Minimum 8 characters"
                required
              />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Replication Mode *</label>
                <select
                  name="replication_mode"
                  value={formData.replication_mode}
                  onChange={handleInputChange}
                >
                  <option value="async">Async (Better Performance)</option>
                  <option value="sync">Sync (Better Consistency)</option>
                </select>
              </div>

              <div className="form-group">
                <label>Namespace</label>
                <input
                  type="text"
                  name="namespace"
                  value={formData.namespace}
                  onChange={handleInputChange}
                  placeholder="e.g., production, staging"
                />
              </div>
            </div>
          </div>
        );

      case 2:
        return (
          <div className="wizard-form">
            <div className="form-group">
              <label>CPU per Node (cores) *</label>
              <input
                type="number"
                name="cpu_per_node"
                value={formData.cpu_per_node}
                onChange={handleInputChange}
                min="1"
                max="32"
                required
              />
              <small>Total: {formData.cpu_per_node * formData.node_count} cores</small>
            </div>

            <div className="form-group">
              <label>Memory per Node (MB) *</label>
              <input
                type="number"
                name="memory_per_node"
                value={formData.memory_per_node}
                onChange={handleInputChange}
                min="512"
                step="512"
                required
              />
              <small>Total: {((formData.memory_per_node * formData.node_count) / 1024).toFixed(1)} GB</small>
            </div>

            <div className="form-group">
              <label>Storage per Node (GB) *</label>
              <input
                type="number"
                name="storage_per_node"
                value={formData.storage_per_node}
                onChange={handleInputChange}
                min="5"
                required
              />
              <small>Total: {formData.storage_per_node * formData.node_count} GB</small>
            </div>

            <div className="resource-summary">
              <h4>Resource Summary</h4>
              <div className="summary-grid">
                <div className="summary-item">
                  <span className="label">Total CPU:</span>
                  <span className="value">{formData.cpu_per_node * formData.node_count} cores</span>
                </div>
                <div className="summary-item">
                  <span className="label">Total Memory:</span>
                  <span className="value">{((formData.memory_per_node * formData.node_count) / 1024).toFixed(1)} GB</span>
                </div>
                <div className="summary-item">
                  <span className="label">Total Storage:</span>
                  <span className="value">{formData.storage_per_node * formData.node_count} GB</span>
                </div>
              </div>
            </div>
          </div>
        );

      case 3:
        return (
          <div className="wizard-form">
            <div className="config-section">
              <h4>Backup Configuration</h4>
              <div className="form-group checkbox-group">
                <label>
                  <input
                    type="checkbox"
                    name="enable_backup"
                    checked={formData.enable_backup}
                    onChange={handleInputChange}
                  />
                  <span>Enable Automatic Backups</span>
                </label>
              </div>

              {formData.enable_backup && (
                <>
                  <div className="form-row">
                    <div className="form-group">
                      <label>Backup Retention (days)</label>
                      <input
                        type="number"
                        name="backup_retention"
                        value={formData.backup_retention}
                        onChange={handleInputChange}
                        min="1"
                        max="30"
                      />
                    </div>
                    <div className="form-group">
                      <label>Backup Schedule (Cron)</label>
                      <input
                        type="text"
                        name="backup_schedule"
                        value={formData.backup_schedule}
                        onChange={handleInputChange}
                        placeholder="0 2 * * *"
                      />
                      <small>Default: Daily at 2 AM</small>
                    </div>
                  </div>
                </>
              )}
            </div>

            <div className="config-section">
              <h4>HAProxy Load Balancer</h4>
              <div className="form-group checkbox-group">
                <label>
                  <input
                    type="checkbox"
                    name="enable_haproxy"
                    checked={formData.enable_haproxy}
                    onChange={handleInputChange}
                  />
                  <span>Enable HAProxy</span>
                </label>
              </div>

              {formData.enable_haproxy && (
                <div className="form-row">
                  <div className="form-group">
                    <label>Write Port</label>
                    <input
                      type="number"
                      name="haproxy_port"
                      value={formData.haproxy_port}
                      onChange={handleInputChange}
                      min="1024"
                      max="65535"
                    />
                  </div>
                  <div className="form-group">
                    <label>Read Port</label>
                    <input
                      type="number"
                      name="haproxy_read_port"
                      value={formData.haproxy_read_port}
                      onChange={handleInputChange}
                      min="1024"
                      max="65535"
                    />
                  </div>
                </div>
              )}
            </div>

            <div className="config-section">
              <h4>Replication Settings</h4>
              <div className="form-group">
                <label>Max Replication Lag (bytes)</label>
                <input
                  type="number"
                  name="max_replication_lag"
                  value={formData.max_replication_lag}
                  onChange={handleInputChange}
                  min="0"
                />
                <small>{(formData.max_replication_lag / 1048576).toFixed(1)} MB</small>
              </div>

              <div className="form-row">
                <div className="form-group checkbox-group">
                  <label>
                    <input
                      type="checkbox"
                      name="use_pg_rewind"
                      checked={formData.use_pg_rewind}
                      onChange={handleInputChange}
                    />
                    <span>Use pg_rewind</span>
                  </label>
                </div>
                <div className="form-group checkbox-group">
                  <label>
                    <input
                      type="checkbox"
                      name="use_slots"
                      checked={formData.use_slots}
                      onChange={handleInputChange}
                    />
                    <span>Use Replication Slots</span>
                  </label>
                </div>
              </div>
            </div>
          </div>
        );

      case 4:
        return (
          <div className="wizard-review">
            <h3>Review Configuration</h3>
            
            <div className="review-section">
              <h4>Basic Information</h4>
              <div className="review-item">
                <span>Cluster Name:</span>
                <strong>{formData.cluster_name}</strong>
              </div>
              <div className="review-item">
                <span>PostgreSQL Version:</span>
                <strong>{formData.postgres_version}</strong>
              </div>
              <div className="review-item">
                <span>Nodes:</span>
                <strong>{formData.node_count} (1 primary + {formData.node_count - 1} replicas)</strong>
              </div>
              <div className="review-item">
                <span>Replication Mode:</span>
                <strong className={formData.replication_mode === 'sync' ? 'text-success' : 'text-warning'}>
                  {formData.replication_mode.toUpperCase()}
                </strong>
              </div>
            </div>

            <div className="review-section">
              <h4>Resources</h4>
              <div className="review-item">
                <span>Total CPU:</span>
                <strong>{formData.cpu_per_node * formData.node_count} cores</strong>
              </div>
              <div className="review-item">
                <span>Total Memory:</span>
                <strong>{((formData.memory_per_node * formData.node_count) / 1024).toFixed(1)} GB</strong>
              </div>
              <div className="review-item">
                <span>Total Storage:</span>
                <strong>{formData.storage_per_node * formData.node_count} GB</strong>
              </div>
            </div>

            <div className="review-section">
              <h4>Features</h4>
              <div className="review-item">
                <span>Automatic Backup:</span>
                <strong className={formData.enable_backup ? 'text-success' : 'text-muted'}>
                  {formData.enable_backup ? `Enabled (${formData.backup_retention} days)` : 'Disabled'}
                </strong>
              </div>
              <div className="review-item">
                <span>HAProxy Load Balancer:</span>
                <strong className={formData.enable_haproxy ? 'text-success' : 'text-muted'}>
                  {formData.enable_haproxy ? `Enabled (Port: ${formData.haproxy_port})` : 'Disabled'}
                </strong>
              </div>
              <div className="review-item">
                <span>Max Replication Lag:</span>
                <strong>{(formData.max_replication_lag / 1048576).toFixed(1)} MB</strong>
              </div>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="wizard-overlay">
      <div className="wizard-container">
        <div className="wizard-header">
          <h2>Create PostgreSQL Cluster</h2>
          <button className="close-btn" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <div className="wizard-progress">
          {STEPS.map((step, index) => {
            const Icon = step.icon;
            const isActive = currentStep === step.id;
            const isCompleted = currentStep > step.id;
            
            return (
              <React.Fragment key={step.id}>
                <div className={`progress-step ${isActive ? 'active' : ''} ${isCompleted ? 'completed' : ''}`}>
                  <div className="step-icon">
                    {isCompleted ? <Check size={20} /> : <Icon size={20} />}
                  </div>
                  <div className="step-info">
                    <div className="step-number">Step {step.id}</div>
                    <div className="step-title">{step.title}</div>
                  </div>
                </div>
                {index < STEPS.length - 1 && (
                  <div className={`progress-line ${isCompleted ? 'completed' : ''}`} />
                )}
              </React.Fragment>
            );
          })}
        </div>

        <div className="wizard-content">
          {renderStepContent()}
        </div>

        <div className="wizard-footer">
          <button
            className="btn-secondary"
            onClick={currentStep === 1 ? onClose : handleBack}
            disabled={isSubmitting}
          >
            {currentStep === 1 ? (
              'Cancel'
            ) : (
              <>
                <ChevronLeft size={16} />
                Back
              </>
            )}
          </button>

          {currentStep < STEPS.length ? (
            <button className="btn-primary" onClick={handleNext}>
              Next
              <ChevronRight size={16} />
            </button>
          ) : (
            <button
              className="btn-primary"
              onClick={handleSubmit}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Creating...' : 'Create Cluster'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default CreatePostgreSQLClusterWizard;
