import React, { useState, useEffect } from 'react';
import { LineChart, Line, AreaChart, Area, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { stackAPI } from '../../api';
import toast from 'react-hot-toast';
import './StackMetricsTab.css';

const StackMetricsTab = ({ stackId }) => {
  const [metrics, setMetrics] = useState(null);
  const [timeRange, setTimeRange] = useState('24h');
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    loadMetrics();
    
    let interval;
    if (autoRefresh) {
      interval = setInterval(loadMetrics, 30000); // Refresh every 30s
    }
    
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [stackId, timeRange, autoRefresh]);

  const loadMetrics = async () => {
    try {
      setLoading(true);
      const response = await stackAPI.getMetrics(stackId, timeRange);
      setMetrics(response.data);
    } catch (error) {
      console.error('Error loading metrics:', error);
      // Mock data for demo
      setMetrics(generateMockMetrics());
    } finally {
      setLoading(false);
    }
  };

  const generateMockMetrics = () => {
    const now = Date.now();
    const data = [];
    for (let i = 23; i >= 0; i--) {
      data.push({
        time: new Date(now - i * 3600000).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }),
        cpu: 30 + Math.random() * 40,
        memory: 50 + Math.random() * 30,
        qps: 2000 + Math.random() * 1000,
        latency: 20 + Math.random() * 40,
        requests: 1000 + Math.random() * 500,
        errors: Math.random() * 50
      });
    }
    return {
      cpu: data,
      memory: data,
      database: data,
      nginx: data
    };
  };

  if (loading) {
    return <div className="loading-metrics">Loading metrics...</div>;
  }

  return (
    <div className="stack-metrics-tab">
      <div className="metrics-header">
        <h3>Stack Metrics</h3>
        <div className="metrics-controls">
          <select value={timeRange} onChange={(e) => setTimeRange(e.target.value)}>
            <option value="1h">Last 1 hour</option>
            <option value="6h">Last 6 hours</option>
            <option value="24h">Last 24 hours</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </select>
          <label className="auto-refresh">
            <input 
              type="checkbox" 
              checked={autoRefresh} 
              onChange={(e) => setAutoRefresh(e.target.checked)} 
            />
            <span>Auto-refresh (30s)</span>
          </label>
        </div>
      </div>

      {/* Resource Usage */}
      <div className="metrics-section">
        <h4>Aggregated Resource Usage</h4>
        <div className="chart-container">
          <h5>CPU Usage (%)</h5>
          <ResponsiveContainer width="100%" height={250}>
            <AreaChart data={metrics?.cpu || []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis domain={[0, 100]} />
              <Tooltip />
              <Legend />
              <Area type="monotone" dataKey="cpu" stroke="#3B82F6" fill="#3B82F6" fillOpacity={0.3} name="CPU %" />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-container">
          <h5>Memory Usage (%)</h5>
          <ResponsiveContainer width="100%" height={250}>
            <AreaChart data={metrics?.memory || []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis domain={[0, 100]} />
              <Tooltip />
              <Legend />
              <Area type="monotone" dataKey="memory" stroke="#10B981" fill="#10B981" fillOpacity={0.3} name="Memory %" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Database Performance */}
      <div className="metrics-section">
        <h4>Database Performance</h4>
        <div className="stats-row">
          <div className="stat-box">
            <div className="stat-label">Current QPS</div>
            <div className="stat-value">2,500</div>
          </div>
          <div className="stat-box">
            <div className="stat-label">Peak QPS</div>
            <div className="stat-value">3,200</div>
          </div>
          <div className="stat-box">
            <div className="stat-label">Avg QPS</div>
            <div className="stat-value">2,100</div>
          </div>
          <div className="stat-box">
            <div className="stat-label">Active Connections</div>
            <div className="stat-value">45/100</div>
          </div>
        </div>

        <div className="chart-container">
          <h5>Query Throughput (QPS)</h5>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={metrics?.database || []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="qps" stroke="#8B5CF6" strokeWidth={2} name="Queries/sec" />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className="chart-container">
          <h5>Query Latency (P95, ms)</h5>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={metrics?.database || []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="latency" stroke="#F59E0B" strokeWidth={2} name="Latency (ms)" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Nginx Traffic */}
      <div className="metrics-section">
        <h4>Nginx Traffic</h4>
        <div className="stats-row">
          <div className="stat-box">
            <div className="stat-label">Current req/s</div>
            <div className="stat-value">1,250</div>
          </div>
          <div className="stat-box">
            <div className="stat-label">Peak req/s</div>
            <div className="stat-value">1,800</div>
          </div>
          <div className="stat-box">
            <div className="stat-label">Success Rate</div>
            <div className="stat-value success">98.5%</div>
          </div>
          <div className="stat-box">
            <div className="stat-label">Error Rate</div>
            <div className="stat-value error">1.5%</div>
          </div>
        </div>

        <div className="chart-container">
          <h5>Requests & Errors</h5>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={metrics?.nginx || []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey="requests" fill="#10B981" name="Requests" />
              <Bar dataKey="errors" fill="#EF4444" name="Errors" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
};

export default StackMetricsTab;
