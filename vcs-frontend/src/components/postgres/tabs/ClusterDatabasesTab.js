import React, { useState, useEffect } from 'react';
import { Database, Plus, Trash2, RefreshCw, Table, Search, Play, Download } from 'lucide-react';
import { clusterAPI } from '../../../api';
import toast from 'react-hot-toast';
import DatabaseExplorer from './DatabaseExplorer';
import './ClusterDatabasesTab.css';

const ClusterDatabasesTab = ({ clusterId, cluster }) => {
  const [databases, setDatabases] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [selectedDatabase, setSelectedDatabase] = useState(null);
  const [formData, setFormData] = useState({
    database_name: '',
    owner: 'postgres',
    encoding: 'UTF8'
  });

  useEffect(() => {
    loadDatabases();
  }, [clusterId]);

  const loadDatabases = async () => {
    try {
      setLoading(true);
      const response = await clusterAPI.listDatabases(clusterId);
      setDatabases(response.data.data || []);
    } catch (error) {
      console.error('Failed to load databases:', error);
      toast.error('Failed to load databases');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateDatabase = async (e) => {
    e.preventDefault();
    try {
      await clusterAPI.createDatabase(clusterId, formData);
      toast.success('Database created successfully');
      setShowCreateForm(false);
      setFormData({ database_name: '', owner: 'postgres', encoding: 'UTF8' });
      loadDatabases();
    } catch (error) {
      console.error('Failed to create database:', error);
      toast.error(error.response?.data?.message || 'Failed to create database');
    }
  };

  const handleDeleteDatabase = async (dbName) => {
    if (!window.confirm(`Are you sure you want to delete database "${dbName}"?`)) {
      return;
    }

    try {
      await clusterAPI.deleteDatabase(clusterId, dbName);
      toast.success('Database deleted successfully');
      loadDatabases();
    } catch (error) {
      console.error('Failed to delete database:', error);
      toast.error('Failed to delete database');
    }
  };

  const handleExploreDatabase = (database) => {
    setSelectedDatabase(database);
  };

  if (selectedDatabase) {
    return (
      <DatabaseExplorer
        clusterId={clusterId}
        database={selectedDatabase}
        onBack={() => setSelectedDatabase(null)}
      />
    );
  }

  return (
    <div className="cluster-databases-tab">
      <div className="databases-header">
        <h2>Databases</h2>
        <div className="header-actions">
          <button className="btn-secondary" onClick={loadDatabases}>
            <RefreshCw size={16} />
            Refresh
          </button>
          <button className="btn-primary" onClick={() => setShowCreateForm(true)}>
            <Plus size={16} />
            Create Database
          </button>
        </div>
      </div>

      {showCreateForm && (
        <div className="create-form-overlay">
          <div className="create-form-modal">
            <div className="modal-header">
              <h3>Create New Database</h3>
              <button className="close-btn" onClick={() => setShowCreateForm(false)}>×</button>
            </div>
            <form onSubmit={handleCreateDatabase}>
              <div className="form-group">
                <label>Database Name *</label>
                <input
                  type="text"
                  value={formData.database_name}
                  onChange={(e) => setFormData({ ...formData, database_name: e.target.value })}
                  placeholder="my_database"
                  required
                />
              </div>
              <div className="form-group">
                <label>Owner</label>
                <input
                  type="text"
                  value={formData.owner}
                  onChange={(e) => setFormData({ ...formData, owner: e.target.value })}
                  placeholder="postgres"
                />
              </div>
              <div className="form-group">
                <label>Encoding</label>
                <select
                  value={formData.encoding}
                  onChange={(e) => setFormData({ ...formData, encoding: e.target.value })}
                >
                  <option value="UTF8">UTF8</option>
                  <option value="LATIN1">LATIN1</option>
                  <option value="SQL_ASCII">SQL_ASCII</option>
                </select>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn-secondary" onClick={() => setShowCreateForm(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  Create Database
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {loading ? (
        <div className="databases-loading">
          <div className="spinner"></div>
          <p>Loading databases...</p>
        </div>
      ) : (
        <div className="databases-grid">
          {databases.map(db => (
            <div key={db.name} className="database-card">
              <div className="database-header">
                <div className="database-icon">
                  <Database size={24} />
                </div>
                <div className="database-info">
                  <h3>{db.name}</h3>
                  <span className="database-owner">Owner: {db.owner}</span>
                </div>
              </div>

              <div className="database-stats">
                <div className="stat-item">
                  <span className="stat-label">Size</span>
                  <span className="stat-value">{db.size || '0 MB'}</span>
                </div>
                <div className="stat-item">
                  <span className="stat-label">Encoding</span>
                  <span className="stat-value">{db.encoding}</span>
                </div>
                <div className="stat-item">
                  <span className="stat-label">Created</span>
                  <span className="stat-value">{new Date(db.created_at).toLocaleDateString()}</span>
                </div>
              </div>

              <div className="database-actions">
                <button
                  className="btn-primary btn-sm"
                  onClick={() => handleExploreDatabase(db)}
                >
                  <Table size={14} />
                  Explore Tables
                </button>
                {db.name !== 'postgres' && db.name !== 'template0' && db.name !== 'template1' && (
                  <button
                    className="btn-danger btn-sm"
                    onClick={() => handleDeleteDatabase(db.name)}
                  >
                    <Trash2 size={14} />
                  </button>
                )}
              </div>
            </div>
          ))}

          {databases.length === 0 && (
            <div className="empty-state">
              <Database size={48} />
              <h3>No Databases Found</h3>
              <p>Create your first database to get started</p>
              <button className="btn-primary" onClick={() => setShowCreateForm(true)}>
                <Plus size={16} />
                Create Database
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ClusterDatabasesTab;
