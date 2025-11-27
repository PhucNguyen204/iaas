import React, { useState, useEffect } from 'react';
import { Search, Download, RefreshCw, Database, Globe, Container } from 'lucide-react';
import { stackAPI, clusterAPI, nginxAPI, dockerAPI } from '../../api';
import './StackLogsTab.css';

const StackLogsTab = ({ stackId }) => {
  const [stack, setStack] = useState(null);
  const [resourceLogs, setResourceLogs] = useState({});
  const [loading, setLoading] = useState(true);
  const [selectedResource, setSelectedResource] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadStackAndLogs();
  }, [stackId]);

  const loadStackAndLogs = async () => {
    try {
      setLoading(true);
      const stackResponse = await stackAPI.getById(stackId);
      const stackData = stackResponse.data?.data || stackResponse.data;
      setStack(stackData);
      
      // Select first resource by default
      if (stackData.resources?.length > 0 && !selectedResource) {
        setSelectedResource(stackData.resources[0]);
        await loadResourceLogs(stackData.resources[0]);
      }
    } catch (error) {
      console.error('Error loading stack:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadResourceLogs = async (resource) => {
    if (!resource) return;
    
    const key = resource.infrastructure_id || resource.outputs?.cluster_id;
    if (!key) return;

    try {
      let response;
      let logData;
      
      switch (resource.resource_type) {
        case 'POSTGRES_CLUSTER':
          const clusterId = resource.outputs?.cluster_id || resource.infrastructure_id;
          response = await clusterAPI.getLogs(clusterId, 100);
          // Backend returns: { data: { cluster_id, logs: [{ node_name, timestamp, logs }] } }
          logData = response.data?.data?.logs || response.data?.logs || [];
          break;
        case 'NGINX_GATEWAY':
          response = await nginxAPI.getLogs(resource.infrastructure_id, 100);
          logData = response.data?.data || response.data;
          break;
        case 'DOCKER_SERVICE':
          response = await dockerAPI.getLogs(resource.infrastructure_id, 100);
          // Backend returns: { data: { logs: ["line1", "line2"] } }
          logData = response.data?.data?.logs || response.data?.logs || [];
          break;
        default:
          return;
      }
      
      setResourceLogs(prev => ({
        ...prev,
        [key]: logData
      }));
    } catch (error) {
      console.error('Error loading logs for resource:', error);
      setResourceLogs(prev => ({
        ...prev,
        [key]: { error: error.response?.data?.message || error.message }
      }));
    }
  };

  const handleResourceSelect = async (resource) => {
    setSelectedResource(resource);
    const key = resource.infrastructure_id || resource.outputs?.cluster_id;
    if (!resourceLogs[key]) {
      await loadResourceLogs(resource);
    }
  };

  const handleRefreshLogs = async () => {
    if (selectedResource) {
      await loadResourceLogs(selectedResource);
    }
  };

  const getResourceIcon = (type) => {
    switch (type?.toUpperCase()) {
      case 'POSTGRES_CLUSTER': return <Database size={16} />;
      case 'NGINX_GATEWAY': return <Globe size={16} />;
      case 'DOCKER_SERVICE': return <Container size={16} />;
      default: return null;
    }
  };

  const downloadLogs = () => {
    if (!selectedResource) return;
    
    const key = selectedResource.infrastructure_id || selectedResource.outputs?.cluster_id;
    const logs = resourceLogs[key];
    if (!logs) return;
    
    let content = '';
    if (Array.isArray(logs)) {
      if (selectedResource.resource_type === 'POSTGRES_CLUSTER') {
        // Cluster logs: array of { node_name, logs }
        content = logs.map(nodeLog => 
          `=== ${nodeLog.node_name} ===\n${nodeLog.logs || 'No logs'}\n`
        ).join('\n');
      } else {
        // Docker logs: array of strings
        content = logs.join('\n');
      }
    } else if (typeof logs === 'string') {
      content = logs;
    }
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${selectedResource.resource_name}-logs-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const renderLogs = () => {
    if (!selectedResource) return <div className="no-logs">Select a resource to view logs</div>;
    
    const key = selectedResource.infrastructure_id || selectedResource.outputs?.cluster_id;
    const logs = resourceLogs[key];
    
    if (!logs) return <div className="loading-logs">Loading logs...</div>;
    if (logs.error) return <div className="error-logs">Error: {logs.error}</div>;
    
    // Filter by search query
    const filterLog = (text) => {
      if (!searchQuery) return true;
      return text?.toLowerCase().includes(searchQuery.toLowerCase());
    };
    
    // Handle cluster logs (array of { node_name, timestamp, logs })
    if (selectedResource.resource_type === 'POSTGRES_CLUSTER' && Array.isArray(logs)) {
      if (logs.length === 0) return <div className="no-logs">No logs available</div>;
      
      return (
        <div className="cluster-logs">
          {logs.map((nodeLog, idx) => (
            <div key={idx} className="node-log-section">
              <div className="node-log-header">
                <Database size={14} />
                <span className="node-name">{nodeLog.node_name}</span>
                {nodeLog.timestamp && <span className="timestamp">{nodeLog.timestamp}</span>}
              </div>
              <pre className="log-content">
                {nodeLog.logs?.split('\n').filter(filterLog).join('\n') || 'No logs'}
              </pre>
            </div>
          ))}
        </div>
      );
    }
    
    // Handle docker/nginx logs (array of strings or single string)
    if (Array.isArray(logs)) {
      const filtered = logs.filter(filterLog);
      if (filtered.length === 0) return <div className="no-logs">No logs matching filter</div>;
      return <pre className="log-content">{filtered.join('\n')}</pre>;
    }
    
    if (typeof logs === 'string') {
      const lines = logs.split('\n').filter(filterLog);
      return <pre className="log-content">{lines.join('\n')}</pre>;
    }
    
    return <pre className="log-content">{JSON.stringify(logs, null, 2)}</pre>;
  };

  if (loading && !stack) {
    return <div className="stack-logs-tab"><div className="loading-logs">Loading...</div></div>;
  }

  const resources = stack?.resources || [];

  return (
    <div className="stack-logs-tab">
      <div className="logs-header">
        <h3>Resource Logs</h3>
        <p className="logs-description">View Docker container logs for each infrastructure resource</p>
      </div>

      <div className="logs-layout">
        {/* Resource Selector Sidebar */}
        <div className="resource-sidebar">
          <h4>Resources</h4>
          {resources.length === 0 ? (
            <div className="no-resources">No resources in stack</div>
          ) : (
            <ul className="resource-list">
              {resources.map((resource, idx) => (
                <li 
                  key={idx}
                  className={`resource-item ${selectedResource?.infrastructure_id === resource.infrastructure_id ? 'selected' : ''}`}
                  onClick={() => handleResourceSelect(resource)}
                >
                  {getResourceIcon(resource.resource_type)}
                  <span className="resource-name">{resource.resource_name}</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Logs Panel */}
        <div className="logs-panel">
          <div className="logs-toolbar">
            <div className="search-box">
              <Search size={16} />
              <input
                type="text"
                placeholder="Filter logs..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <button className="btn btn-secondary btn-sm" onClick={handleRefreshLogs}>
              <RefreshCw size={14} /> Refresh
            </button>
            <button className="btn btn-secondary btn-sm" onClick={downloadLogs}>
              <Download size={14} /> Download
            </button>
          </div>
          
          <div className="logs-container">
            {renderLogs()}
          </div>
        </div>
      </div>
    </div>
  );
};

export default StackLogsTab;
