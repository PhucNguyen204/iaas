import React, { useState, useEffect } from 'react';
import { Plus, Search } from 'lucide-react';
import { stackAPI } from '../api';
import Layout from './common/Layout';
import StatusBadge from './common/StatusBadge';
import ResourceIcon from './common/ResourceIcon';
import CreateStackModal from './CreateStackModal';
import toast from 'react-hot-toast';
import './StackDashboard.css';

const StackDashboard = ({ onLogout, onViewStack }) => {
  const [stacks, setStacks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterEnvironment, setFilterEnvironment] = useState('all');
  const [filterStatus, setFilterStatus] = useState('all');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [stats, setStats] = useState({
    totalStacks: 0,
    runningStacks: 0,
    issues: 0,
    totalCost: 0,
    databases: 0,
    gateways: 0,
    services: 0,
    uptime: 99.8
  });

  useEffect(() => {
    loadStacks();
    
    // Auto-refresh every 10 seconds to update stack statuses
    const interval = setInterval(() => {
      loadStacks();
    }, 10000);
    
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadStacks = async () => {
    try {
      setLoading(true);
      const response = await stackAPI.getAll();
      console.log('Stack API response:', response.data);
      // API returns { success, code, message, data: { stacks: [...], total_count, ... } }
      const stacksData = response.data?.data?.stacks || response.data?.stacks || [];
      console.log('Parsed stacks:', stacksData);
      setStacks(stacksData);
      calculateStats(stacksData);
    } catch (error) {
      // Don't show error if it's 401 (handled by interceptor)
      if (error.response?.status !== 401) {
        console.error('Error loading stacks:', error);
        console.error('Error response:', error.response?.data);
        toast.error('Failed to load stacks');
      }
      setStacks([]);
    } finally {
      setLoading(false);
    }
  };

  const calculateStats = (stacksData) => {
    const running = stacksData.filter(s => s.status === 'running').length;
    const issues = stacksData.filter(s => s.status === 'degraded' || s.status === 'failed').length;
    
    let databases = 0, gateways = 0, services = 0;
    stacksData.forEach(stack => {
      if (stack.resources) {
        stack.resources.forEach(r => {
          if (r.resource_type?.includes('POSTGRES')) databases++;
          if (r.resource_type?.includes('NGINX')) gateways++;
          if (r.resource_type?.includes('DOCKER')) services++;
        });
      }
    });

    setStats({
      totalStacks: stacksData.length,
      runningStacks: running,
      issues: issues,
      totalCost: 1245,
      databases,
      gateways,
      services,
      uptime: 99.8
    });
  };

  const filteredStacks = stacks.filter(stack => {
    const matchesSearch = stack.name?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesEnv = filterEnvironment === 'all' || stack.environment === filterEnvironment;
    const matchesStatus = filterStatus === 'all' || stack.status === filterStatus;
    return matchesSearch && matchesEnv && matchesStatus;
  });

  const handleCreateStack = () => {
    setShowCreateModal(true);
  };

  const handleCreateSuccess = () => {
    loadStacks();
  };

  return (
    <Layout onLogout={onLogout} activeTab="stacks">
      <div className="stack-dashboard">
        {/* System Overview */}
        <div className="system-overview">
          <h2>System Overview</h2>
          <div className="stats-grid">
            <div className="stat-card">
              <ResourceIcon type="stack" size={24} />
              <div className="stat-value">{stats.totalStacks}</div>
              <div className="stat-label">Total Stacks</div>
            </div>
            <div className="stat-card stat-success">
              <div className="stat-value">{stats.runningStacks}</div>
              <div className="stat-label">Running</div>
            </div>
            <div className="stat-card stat-warning">
              <div className="stat-value">{stats.issues}</div>
              <div className="stat-label">Issues</div>
            </div>
            <div className="stat-card">
              <div className="stat-value">${stats.totalCost}</div>
              <div className="stat-label">Cost/month</div>
            </div>
            <div className="stat-card">
              <ResourceIcon type="database" size={24} />
              <div className="stat-value">{stats.databases}</div>
              <div className="stat-label">Databases</div>
            </div>
            <div className="stat-card">
              <ResourceIcon type="gateway" size={24} />
              <div className="stat-value">{stats.gateways}</div>
              <div className="stat-label">Gateways</div>
            </div>
            <div className="stat-card">
              <ResourceIcon type="container" size={24} />
              <div className="stat-value">{stats.services}</div>
              <div className="stat-label">Services</div>
            </div>
            <div className="stat-card stat-success">
              <div className="stat-value">{stats.uptime}%</div>
              <div className="stat-label">Uptime (30d)</div>
            </div>
          </div>
        </div>

        {/* Search and Filters */}
        <div className="toolbar">
          <div className="search-box">
            <Search size={20} />
            <input
              type="text"
              placeholder="Search stacks..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          <div className="filters">
            <select 
              value={filterEnvironment} 
              onChange={(e) => setFilterEnvironment(e.target.value)}
            >
              <option value="all">All Environments</option>
              <option value="production">Production</option>
              <option value="staging">Staging</option>
              <option value="development">Development</option>
            </select>
            <select 
              value={filterStatus} 
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <option value="all">All Status</option>
              <option value="running">Running</option>
              <option value="stopped">Stopped</option>
              <option value="degraded">Degraded</option>
              <option value="failed">Failed</option>
            </select>
          </div>
          <button className="btn btn-primary" onClick={handleCreateStack}>
            <Plus size={20} />
            New Stack
          </button>
        </div>

        {/* Stacks Grid */}
        {loading ? (
          <div className="loading-state">Loading stacks...</div>
        ) : filteredStacks.length === 0 ? (
          <div className="empty-state">
            <ResourceIcon type="stack" size={48} />
            <h3>No stacks found</h3>
            <p>Create your first stack to get started</p>
          </div>
        ) : (
          <div className="stacks-grid">
            {filteredStacks.map(stack => (
              <StackCard key={stack.id} stack={stack} onRefresh={loadStacks} onViewStack={onViewStack} />
            ))}
          </div>
        )}
      </div>

      <CreateStackModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={handleCreateSuccess}
      />
    </Layout>
  );
};

const StackCard = ({ stack, onRefresh, onViewStack }) => {
  const [loading, setLoading] = useState(false);

  const handleAction = async (action) => {
    try {
      setLoading(true);
      await stackAPI[action](stack.id);
      toast.success(`Stack ${action} successful`);
      onRefresh();
    } catch (error) {
      toast.error(`Failed to ${action} stack`);
    } finally {
      setLoading(false);
    }
  };

  const resourceCount = stack.resources?.length || 0;
  const runningResources = stack.resources?.filter(r => r.status === 'running').length || 0;

  return (
    <div className={`stack-card ${stack.status}`}>
      <div className="stack-header">
        <div className="stack-title">
          <ResourceIcon type="stack" size={24} />
          <h3>{stack.name}</h3>
        </div>
        <StatusBadge status={stack.status} />
      </div>

      <p className="stack-description">{stack.description || 'No description'}</p>

      <div className="stack-tags">
        {stack.tags && Object.keys(stack.tags).map(key => (
          <span key={key} className="tag">#{stack.tags[key]}</span>
        ))}
        {stack.environment && <span className="tag env-tag">{stack.environment}</span>}
      </div>

      <div className="stack-meta">
        <span>Created: {new Date(stack.created_at).toLocaleDateString()}</span>
        <span>Updated: {new Date(stack.updated_at).toLocaleDateString()}</span>
      </div>

      <div className="stack-resources">
        <h4>Resources ({resourceCount})</h4>
        <div className="resource-list">
          {stack.resources?.slice(0, 5).map((resource, idx) => (
            <div key={idx} className="resource-item">
              <ResourceIcon type={resource.resource_type} size={16} />
              <span>{resource.resource_name || resource.resource_type}</span>
              <StatusBadge status={resource.status} />
            </div>
          ))}
          {resourceCount > 5 && (
            <div className="resource-item-more">+{resourceCount - 5} more</div>
          )}
        </div>
      </div>

      <div className="stack-stats">
        <div className="stat-item">
          <span className="label">Status:</span>
          <span className="value">{runningResources}/{resourceCount} Running</span>
        </div>
      </div>

      <div className="stack-actions">
        <button 
          className="btn btn-secondary" 
          disabled={loading}
          onClick={() => onViewStack && onViewStack(stack.id)}
        >
          View Details
        </button>
        <button className="btn btn-secondary" disabled={loading}>
          Logs
        </button>
        <button 
          className="btn btn-icon" 
          onClick={() => handleAction('restart')}
          disabled={loading}
        >
          ⋮
        </button>
      </div>
    </div>
  );
};

export default StackDashboard;
