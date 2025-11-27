import React from 'react';
import './StatusBadge.css';

const StatusBadge = ({ status }) => {
  const getStatusConfig = (status) => {
    switch (status?.toLowerCase()) {
      case 'running':
      case 'healthy':
      case 'active':
        return { color: 'green', label: 'RUNNING' };
      case 'warning':
      case 'degraded':
        return { color: 'yellow', label: 'WARNING' };
      case 'failed':
      case 'error':
      case 'down':
        return { color: 'red', label: 'FAILED' };
      case 'stopped':
      case 'inactive':
        return { color: 'gray', label: 'STOPPED' };
      case 'deleted':
      case 'removed':
      case 'terminated':
        return { color: 'red', label: 'DELETED' };
      case 'starting':
      case 'initializing':
      case 'creating':
        return { color: 'blue', label: 'STARTING' };
      case 'pending':
        return { color: 'orange', label: 'PENDING' };
      default:
        return { color: 'gray', label: status?.toUpperCase() || 'UNKNOWN' };
    }
  };

  const config = getStatusConfig(status);

  return (
    <span className={`status-badge status-${config.color}`}>
      <span className="status-dot"></span>
      {config.label}
    </span>
  );
};

export default StatusBadge;
