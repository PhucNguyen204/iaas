import React, { useState, useEffect } from 'react';
import { ArrowLeft, Database, Activity, RefreshCw, Power, StopCircle, Play, MoreVertical, Users, Settings, GitBranch, BarChart3, FileText, Server } from 'lucide-react';
import { clusterAPI } from '../../api';
import toast from 'react-hot-toast';
import StatusBadge from '../common/StatusBadge';
import ClusterOverviewTab from './tabs/ClusterOverviewTab';
import ClusterDatabasesTab from './tabs/ClusterDatabasesTab';
import ClusterReplicationTab from './tabs/ClusterReplicationTab';
import ClusterMonitoringTab from './tabs/ClusterMonitoringTab';
import ClusterLogsTab from './tabs/ClusterLogsTab';
import ClusterSettingsTab from './tabs/ClusterSettingsTab';
import ClusterNodesTab from './tabs/ClusterNodesTab';
import './PostgreSQLClusterDetail.css';

const TABS = [
  { id: 'overview', label: 'Overview', icon: Database },
  { id: 'nodes', label: 'Nodes', icon: Activity },
  { id: 'databases', label: 'Databases', icon: Database },
  { id: 'replication', label: 'Replication', icon: GitBranch },
  { id: 'monitoring', label: 'Monitoring', icon: BarChart3 },
  { id: 'logs', label: 'Logs', icon: FileText },
  { id: 'settings', label: 'Settings', icon: Settings },
];

const PostgreSQLClusterDetail = ({ clusterId, onBack }) => {
  const [activeTab, setActiveTab] = useState('overview');
  const [cluster, setCluster] = useState(null);
  const [loading, setLoading] = useState(true);
  const [showMenu, setShowMenu] = useState(false);

  useEffect(() => {
    loadCluster();
  }, [clusterId]);

  const loadCluster = async () => {
    try {
      setLoading(true);
      const response = await clusterAPI.getById(clusterId);
      setCluster(response.data.data);
    } catch (error) {
      console.error('Failed to load cluster:', error);
      toast.error('Failed to load cluster details');
    } finally {
      setLoading(false);
    }
  };

  const handleAction = async (action) => {
    try {
      setShowMenu(false);
      let response;
      
      switch (action) {
        case 'start':
          response = await clusterAPI.start(clusterId);
          toast.success('Cluster started successfully');
          break;
        case 'stop':
          response = await clusterAPI.stop(clusterId);
          toast.success('Cluster stopped successfully');
          break;
        case 'restart':
          response = await clusterAPI.restart(clusterId);
          toast.success('Cluster restarted successfully');
          break;
        case 'delete':
          if (window.confirm('Are you sure you want to delete this cluster? This action cannot be undone.')) {
            await clusterAPI.delete(clusterId);
            toast.success('Cluster deleted successfully');
            onBack();
          }
          return;
        default:
          return;
      }
      
      await loadCluster();
    } catch (error) {
      console.error(`Failed to ${action} cluster:`, error);
      toast.error(`Failed to ${action} cluster`);
    }
  };

  const renderTabContent = () => {
    if (!cluster) return null;

    switch (activeTab) {
      case 'overview':
        return <ClusterOverviewTab cluster={cluster} onRefresh={loadCluster} />;
      case 'nodes':
        return <ClusterNodesTab clusterId={clusterId} cluster={cluster} onRefresh={loadCluster} />;
      case 'databases':
        return <ClusterDatabasesTab clusterId={clusterId} cluster={cluster} />;
      case 'replication':
        return <ClusterReplicationTab clusterId={clusterId} cluster={cluster} />;
      case 'monitoring':
        return <ClusterMonitoringTab clusterId={clusterId} cluster={cluster} />;
      case 'logs':
        return <ClusterLogsTab clusterId={clusterId} cluster={cluster} />;
      case 'settings':
        return <ClusterSettingsTab clusterId={clusterId} cluster={cluster} onUpdate={loadCluster} />;
      default:
        return null;
    }
  };

  if (loading) {
    return (
      <div className="cluster-detail-loading">
        <div className="spinner"></div>
        <p>Loading cluster details...</p>
      </div>
    );
  }

  if (!cluster) {
    return (
      <div className="cluster-detail-error">
        <p>Cluster not found</p>
        <button className="btn-primary" onClick={onBack}>Go Back</button>
      </div>
    );
  }

  return (
    <div className="cluster-detail-page">
      <div className="cluster-detail-header">
        <div className="header-top">
          <button className="back-button" onClick={onBack}>
            <ArrowLeft size={20} />
            Back to Clusters
          </button>

          <div className="header-actions">
            <button className="btn-icon" onClick={loadCluster} title="Refresh">
              <RefreshCw size={18} />
            </button>
            
            <button className="btn-secondary" onClick={() => handleAction('restart')}>
              <Activity size={16} />
              Restart All
            </button>

            <button className="btn-secondary" onClick={() => handleAction('stop')}>
              <StopCircle size={16} />
              Stop All
            </button>

            <div className="dropdown-container">
              <button className="btn-icon" onClick={() => setShowMenu(!showMenu)}>
                <MoreVertical size={18} />
              </button>
              
              {showMenu && (
                <div className="dropdown-menu">
                  <button onClick={() => handleAction('start')}>
                    <Play size={16} />
                    Start All
                  </button>
                  <button onClick={() => handleAction('restart')}>
                    <RefreshCw size={16} />
                    Restart All
                  </button>
                  <button onClick={() => handleAction('stop')}>
                    <StopCircle size={16} />
                    Stop All
                  </button>
                  <div className="dropdown-divider"></div>
                  <button className="danger" onClick={() => handleAction('delete')}>
                    <Power size={16} />
                    Delete Cluster
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="header-info">
          <div className="cluster-title">
            <Database size={28} />
            <div>
              <h1>{cluster.cluster_name}</h1>
              <div className="cluster-meta">
                <span className="meta-item">
                  PostgreSQL {cluster.postgres_version}
                </span>
                <span className="meta-divider">•</span>
                <span className="meta-item">
                  {cluster.nodes?.length || cluster.node_count} Nodes
                </span>
                <span className="meta-divider">•</span>
                <span className="meta-item">
                  {cluster.replication_mode === 'sync' ? 'Synchronous' : 'Asynchronous'} Replication
                </span>
              </div>
            </div>
          </div>

          <div className="cluster-status">
            <StatusBadge status={cluster.status} />
          </div>
        </div>
      </div>

      <div className="cluster-tabs">
        {TABS.map(tab => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              className={`tab-button ${activeTab === tab.id ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              <Icon size={18} />
              {tab.label}
            </button>
          );
        })}
      </div>

      <div className="cluster-content">
        {renderTabContent()}
      </div>
    </div>
  );
};

export default PostgreSQLClusterDetail;
