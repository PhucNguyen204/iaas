import React from 'react';
import ResourceIcon from '../common/ResourceIcon';
import StatusBadge from '../common/StatusBadge';
import './StackTopologyTab.css';

const StackTopologyTab = ({ stack }) => {
  const resources = stack.resources || [];

  // Group resources by role
  const groupedResources = resources.reduce((acc, resource) => {
    const role = resource.role || 'other';
    if (!acc[role]) acc[role] = [];
    acc[role].push(resource);
    return acc;
  }, {});

  return (
    <div className="stack-topology-tab">
      <div className="topology-header">
        <h3>Stack Topology</h3>
        <div className="topology-legend">
          <div className="legend-item">
            <span className="dot dot-green"></span>
            <span>Running</span>
          </div>
          <div className="legend-item">
            <span className="dot dot-yellow"></span>
            <span>Warning</span>
          </div>
          <div className="legend-item">
            <span className="dot dot-red"></span>
            <span>Down</span>
          </div>
          <div className="legend-item">
            <span className="arrow">→</span>
            <span>Data Flow</span>
          </div>
        </div>
      </div>

      <div className="topology-diagram">
        {/* Gateway Layer */}
        {groupedResources.gateway && (
          <div className="topology-layer">
            <div className="layer-label">Gateway</div>
            <div className="nodes-row">
              {groupedResources.gateway.map((resource, idx) => (
                <TopologyNode key={idx} resource={resource} />
              ))}
            </div>
            <div className="connection-line"></div>
          </div>
        )}

        {/* Application Layer */}
        {groupedResources.app && (
          <div className="topology-layer">
            <div className="layer-label">Application</div>
            <div className="nodes-row">
              {groupedResources.app.map((resource, idx) => (
                <TopologyNode key={idx} resource={resource} />
              ))}
            </div>
            <div className="connection-line"></div>
          </div>
        )}

        {/* Cache Layer */}
        {groupedResources.cache && (
          <div className="topology-layer">
            <div className="layer-label">Cache</div>
            <div className="nodes-row">
              {groupedResources.cache.map((resource, idx) => (
                <TopologyNode key={idx} resource={resource} />
              ))}
            </div>
            <div className="connection-line"></div>
          </div>
        )}

        {/* Database Layer */}
        {groupedResources.database && (
          <div className="topology-layer">
            <div className="layer-label">Database</div>
            <div className="nodes-row">
              {groupedResources.database.map((resource, idx) => (
                <TopologyNode key={idx} resource={resource} />
              ))}
            </div>
          </div>
        )}

        {/* Other Resources */}
        {groupedResources.other && (
          <div className="topology-layer">
            <div className="layer-label">Other</div>
            <div className="nodes-row">
              {groupedResources.other.map((resource, idx) => (
                <TopologyNode key={idx} resource={resource} />
              ))}
            </div>
          </div>
        )}

        {resources.length === 0 && (
          <div className="empty-topology">
            <p>No resources to display</p>
          </div>
        )}
      </div>

      {/* Dependencies List */}
      {resources.some(r => r.depends_on && r.depends_on.length > 0) && (
        <div className="dependencies-section">
          <h4>Resource Dependencies</h4>
          <div className="dependencies-list">
            {resources.filter(r => r.depends_on && r.depends_on.length > 0).map((resource, idx) => (
              <div key={idx} className="dependency-item">
                <span className="dependent">{resource.resource_name}</span>
                <span className="arrow">→</span>
                <span className="dependencies">{resource.depends_on.join(', ')}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

const TopologyNode = ({ resource }) => {
  const statusColor = resource.status === 'running' ? 'green' : 
                      resource.status === 'warning' ? 'yellow' : 'red';

  return (
    <div className={`topology-node status-${statusColor}`}>
      <div className="node-icon">
        <ResourceIcon type={resource.resource_type} size={24} />
      </div>
      <div className="node-name">{resource.resource_name || resource.resource_type}</div>
      <div className="node-status">
        <StatusBadge status={resource.status} />
      </div>
    </div>
  );
};

export default StackTopologyTab;
