import React, { useState, useEffect } from 'react';
import { ArrowLeft, Table, Search, Play, Download, ChevronLeft, ChevronRight, RefreshCw, Columns } from 'lucide-react';
import { clusterAPI } from '../../../api';
import toast from 'react-hot-toast';
import './DatabaseExplorer.css';

const DatabaseExplorer = ({ clusterId, database, onBack }) => {
  const [tables, setTables] = useState([]);
  const [selectedTable, setSelectedTable] = useState(null);
  const [tableSchema, setTableSchema] = useState(null);
  const [tableData, setTableData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [dataLoading, setDataLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [limit] = useState(50);
  const [totalRows, setTotalRows] = useState(0);
  const [showSqlEditor, setShowSqlEditor] = useState(false);
  const [sqlQuery, setSqlQuery] = useState('');
  const [queryResult, setQueryResult] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadTables();
  }, [clusterId, database.name]);

  useEffect(() => {
    if (selectedTable) {
      loadTableData();
      loadTableSchema();
    }
  }, [selectedTable, page]);

  const loadTables = async () => {
    try {
      setLoading(true);
      const response = await clusterAPI.getTables(clusterId, database.name);
      setTables(response.data.data || []);
    } catch (error) {
      console.error('Failed to load tables:', error);
      toast.error('Failed to load tables');
    } finally {
      setLoading(false);
    }
  };

  const loadTableSchema = async () => {
    try {
      const response = await clusterAPI.getTableSchema(clusterId, database.name, selectedTable);
      setTableSchema(response.data.data);
    } catch (error) {
      console.error('Failed to load table schema:', error);
      toast.error('Failed to load table schema');
    }
  };

  const loadTableData = async () => {
    try {
      setDataLoading(true);
      const response = await clusterAPI.getTableData(clusterId, database.name, selectedTable, page, limit);
      setTableData(response.data.data.rows || []);
      setTotalRows(response.data.data.total || 0);
    } catch (error) {
      console.error('Failed to load table data:', error);
      toast.error('Failed to load table data');
    } finally {
      setDataLoading(false);
    }
  };

  const executeQuery = async () => {
    if (!sqlQuery.trim()) {
      toast.error('Please enter a SQL query');
      return;
    }

    try {
      const response = await clusterAPI.executeQuery(clusterId, database.name, sqlQuery);
      setQueryResult(response.data.data);
      toast.success('Query executed successfully');
    } catch (error) {
      console.error('Query execution failed:', error);
      toast.error(error.response?.data?.message || 'Query execution failed');
      setQueryResult({ error: error.response?.data?.message || 'Query execution failed' });
    }
  };

  const handleTableSelect = (tableName) => {
    setSelectedTable(tableName);
    setPage(1);
    setShowSqlEditor(false);
    setQueryResult(null);
  };

  const totalPages = Math.ceil(totalRows / limit);

  const filteredTables = tables.filter(table =>
    table.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="database-explorer">
      <div className="explorer-header">
        <button className="back-btn" onClick={onBack}>
          <ArrowLeft size={18} />
          Back to Databases
        </button>
        <div className="database-title">
          <Table size={24} />
          <div>
            <h2>{database.name}</h2>
            <span className="database-meta">{tables.length} tables</span>
          </div>
        </div>
        <button className="btn-primary" onClick={() => setShowSqlEditor(!showSqlEditor)}>
          <Play size={16} />
          SQL Editor
        </button>
      </div>

      <div className="explorer-content">
        <div className="tables-sidebar">
          <div className="sidebar-header">
            <h3>Tables</h3>
            <button className="btn-icon" onClick={loadTables} title="Refresh">
              <RefreshCw size={16} />
            </button>
          </div>

          <div className="search-box">
            <Search size={16} />
            <input
              type="text"
              placeholder="Search tables..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>

          <div className="tables-list">
            {loading ? (
              <div className="sidebar-loading">
                <div className="spinner-sm"></div>
              </div>
            ) : (
              <>
                {filteredTables.map(table => (
                  <button
                    key={table}
                    className={`table-item ${selectedTable === table ? 'active' : ''}`}
                    onClick={() => handleTableSelect(table)}
                  >
                    <Table size={16} />
                    <span>{table}</span>
                  </button>
                ))}
                {filteredTables.length === 0 && (
                  <div className="empty-tables">
                    <p>No tables found</p>
                  </div>
                )}
              </>
            )}
          </div>
        </div>

        <div className="data-panel">
          {showSqlEditor ? (
            <div className="sql-editor">
              <div className="editor-header">
                <h3>SQL Query Editor</h3>
                <button className="btn-primary" onClick={executeQuery}>
                  <Play size={16} />
                  Execute Query
                </button>
              </div>

              <textarea
                className="sql-textarea"
                placeholder="Enter your SQL query here...&#10;Example: SELECT * FROM users WHERE id > 100 LIMIT 10;"
                value={sqlQuery}
                onChange={(e) => setSqlQuery(e.target.value)}
              />

              {queryResult && (
                <div className="query-result">
                  {queryResult.error ? (
                    <div className="error-message">
                      <strong>Error:</strong> {queryResult.error}
                    </div>
                  ) : (
                    <>
                      <div className="result-header">
                        <h4>Query Result</h4>
                        <span className="result-count">
                          {queryResult.rows?.length || 0} rows returned
                        </span>
                      </div>
                      {queryResult.rows && queryResult.rows.length > 0 ? (
                        <div className="result-table-wrapper">
                          <table className="result-table">
                            <thead>
                              <tr>
                                {Object.keys(queryResult.rows[0]).map(col => (
                                  <th key={col}>{col}</th>
                                ))}
                              </tr>
                            </thead>
                            <tbody>
                              {queryResult.rows.map((row, idx) => (
                                <tr key={idx}>
                                  {Object.values(row).map((val, i) => (
                                    <td key={i}>
                                      {val === null ? <span className="null-value">NULL</span> : String(val)}
                                    </td>
                                  ))}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      ) : (
                        <div className="empty-result">
                          <p>Query executed successfully. No rows returned.</p>
                        </div>
                      )}
                    </>
                  )}
                </div>
              )}
            </div>
          ) : selectedTable ? (
            <div className="table-viewer">
              <div className="viewer-header">
                <div className="viewer-title">
                  <h3>{selectedTable}</h3>
                  <span className="row-count">{totalRows.toLocaleString()} rows</span>
                </div>
                <div className="viewer-actions">
                  <button className="btn-secondary btn-sm" onClick={loadTableData}>
                    <RefreshCw size={14} />
                    Refresh
                  </button>
                  <button className="btn-secondary btn-sm" onClick={() => {}}>
                    <Download size={14} />
                    Export
                  </button>
                </div>
              </div>

              {tableSchema && (
                <div className="schema-info">
                  <h4>
                    <Columns size={16} />
                    Table Schema
                  </h4>
                  <div className="schema-columns">
                    {tableSchema.columns?.map(col => (
                      <div key={col.name} className="schema-column">
                        <div className="column-name">
                          <strong>{col.name}</strong>
                          {col.is_primary && <span className="pk-badge">PK</span>}
                        </div>
                        <div className="column-type">{col.data_type}</div>
                        <div className="column-nullable">
                          {col.is_nullable ? 'Nullable' : 'Not Null'}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {dataLoading ? (
                <div className="data-loading">
                  <div className="spinner"></div>
                  <p>Loading data...</p>
                </div>
              ) : (
                <>
                  {tableData.length > 0 ? (
                    <>
                      <div className="data-table-wrapper">
                        <table className="data-table">
                          <thead>
                            <tr>
                              <th className="row-number">#</th>
                              {Object.keys(tableData[0]).map(col => (
                                <th key={col}>{col}</th>
                              ))}
                            </tr>
                          </thead>
                          <tbody>
                            {tableData.map((row, idx) => (
                              <tr key={idx}>
                                <td className="row-number">{(page - 1) * limit + idx + 1}</td>
                                {Object.values(row).map((val, i) => (
                                  <td key={i}>
                                    {val === null ? (
                                      <span className="null-value">NULL</span>
                                    ) : typeof val === 'object' ? (
                                      <code>{JSON.stringify(val)}</code>
                                    ) : (
                                      String(val)
                                    )}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>

                      {totalPages > 1 && (
                        <div className="pagination">
                          <button
                            className="btn-secondary btn-sm"
                            onClick={() => setPage(p => Math.max(1, p - 1))}
                            disabled={page === 1}
                          >
                            <ChevronLeft size={16} />
                            Previous
                          </button>
                          <span className="page-info">
                            Page {page} of {totalPages}
                          </span>
                          <button
                            className="btn-secondary btn-sm"
                            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                            disabled={page === totalPages}
                          >
                            Next
                            <ChevronRight size={16} />
                          </button>
                        </div>
                      )}
                    </>
                  ) : (
                    <div className="empty-data">
                      <Table size={48} />
                      <h3>No Data</h3>
                      <p>This table is empty</p>
                    </div>
                  )}
                </>
              )}
            </div>
          ) : (
            <div className="no-selection">
              <Table size={64} />
              <h3>Select a Table</h3>
              <p>Choose a table from the sidebar to view its data</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default DatabaseExplorer;
