import React, { useState } from 'react';
import { CheckCircle, Eye } from 'lucide-react';
import StatusBadge from '../common/StatusBadge';
import ResourceIcon from '../common/ResourceIcon';
import InfrastructureDetailModal from '../InfrastructureDetailModal';
import './StackOverviewTab.css';

const StackOverviewTab = ({ stack, onRefresh }) => {
  const resources = stack.resources || [];
  const runningCount = resources.filter(r => r.status === 'running').length;
  const totalCount = resources.length;
  const healthPercent = totalCount > 0 ? ((runningCount / totalCount) * 100).toFixed(1) : 0;
  const [selectedResource, setSelectedResource] = useState(null);
  const [showDetailModal, setShowDetailModal] = useState(false);

  const handleViewDetails = (resource) => {
    setSelectedResource(resource);
    setShowDetailModal(true);
  };

  return (
    <div className="stack-overview-tab">
      {/* Stack Health Summary */}
      <div className="health-summary">
        <h3>Stack Health</h3>
        <div className="health-grid">
          <div className="health-item">
            <div className="health-label">Overall Health</div>
            <div className="health-value">
              <CheckCircle size={24} className="icon-success" />
              <span className="success">{healthPercent}%</span>
            </div>
          </div>
          <div className="health-item">
            <div className="health-label">Active Components</div>
            <div className="health-value">
              <span className="value">{runningCount}/{totalCount} Running</span>
            </div>
          </div>
          <div className="health-item">
            <div className="health-label">Resource Usage</div>
            <div className="health-value">
              <span className="value">N/A (Metrics API needed)</span>
            </div>
          </div>
          <div className="health-item">
            <div className="health-label">Active Alerts</div>
            <div className="health-value">
              <span className="value">{resources.filter(r => r.status !== 'running').length > 0 ? `${resources.filter(r => r.status !== 'running').length} Warning` : 'All OK'}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Resources List */}
      <div className="resources-section">
        <h3>Resources ({totalCount})</h3>
        
        {resources.map((resource, idx) => (
          <ResourceCard 
            key={idx} 
            resource={resource} 
            onViewDetails={() => handleViewDetails(resource)}
          />
        ))}

        {totalCount === 0 && (
          <div className="empty-resources">
            <p>No resources in this stack</p>
          </div>
        )}
      </div>

      <InfrastructureDetailModal
        isOpen={showDetailModal}
        onClose={() => {
          setShowDetailModal(false);
          setSelectedResource(null);
        }}
        resource={selectedResource}
        onRefresh={onRefresh}
      />
    </div>
  );
};

const ResourceCard = ({ resource, onViewDetails }) => {
  return (
    <div className="resource-card">
      <div className="resource-header">
        <div className="resource-title">
          <ResourceIcon type={resource.resource_type} size={24} />
          <div>
            <h4>{resource.resource_name || resource.resource_type}</h4>
            <span className="resource-type">{resource.resource_type}</span>
          </div>
        </div>
        <StatusBadge status={resource.status} />
      </div>

      <div className="resource-details">
        <div className="detail-row">
          <span className="label">Role:</span>
          <span className="value">{resource.role || 'N/A'}</span>
        </div>
        {resource.infrastructure_id && (
          <div className="detail-row">
            <span className="label">Infrastructure ID:</span>
            <span className="value code">{resource.infrastructure_id.substring(0, 8)}...</span>
          </div>
        )}
        {resource.depends_on && resource.depends_on.length > 0 && (
          <div className="detail-row">
            <span className="label">Dependencies:</span>
            <span className="value">{resource.depends_on.join(', ')}</span>
          </div>
        )}
      </div>

      {resource.outputs && Object.keys(resource.outputs).length > 0 && (
        <div className="resource-outputs">
          <h5>Quick Info</h5>
          {Object.entries(resource.outputs).slice(0, 3).map(([key, value]) => (
            <div key={key} className="output-item">
              <span className="output-key">{key}:</span>
              <span className="output-value code">{String(value)}</span>
            </div>
          ))}
          {Object.keys(resource.outputs).length > 3 && (
            <div className="output-item">
              <span className="output-key">...</span>
              <span className="output-value">+{Object.keys(resource.outputs).length - 3} more</span>
            </div>
          )}
        </div>
      )}

      <div className="resource-actions">
        <button 
          className="btn btn-secondary btn-sm" 
          onClick={onViewDetails}
        >
          <Eye size={16} />
          View Details
        </button>
      </div>
    </div>
  );
};

export default StackOverviewTab;
