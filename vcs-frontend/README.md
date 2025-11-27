# 🚀 VCS Frontend - Stack-Based Infrastructure Management UI

## 📋 Tổng quan

Frontend React cho hệ thống quản lý infrastructure theo stack-centric design. Giao diện chuyên nghiệp với các tính năng:

- ✅ **Stack Dashboard**: System overview với stats tổng hợp
- ✅ **Stack Cards**: Hiển thị 3 states (Running, Warning, Failed)
- ✅ **Stack Detail Page**: 5 tabs (Overview, Topology, Metrics, Logs, Configuration)
- ✅ **Real-time Charts**: Recharts cho CPU, Memory, QPS, Latency
- ✅ **Unified Logs**: Tích hợp logs từ tất cả resources
- ✅ **Topology Diagram**: Trực quan hóa dependencies

## 🛠️ Công nghệ sử dụng

- **React 18.2.0**: UI framework
- **React Router DOM 6.20.0**: Navigation
- **Axios 1.6.2**: HTTP client
- **Recharts 2.10.3**: Charts & metrics visualization
- **React Hot Toast 2.4.1**: Notifications
- **Lucide React 0.294.0**: Icons

## 📂 Cấu trúc thư mục

```
vcs-frontend/
├── public/
│   └── index.html
├── src/
│   ├── components/
│   │   ├── common/
│   │   │   ├── Layout.js              # Main layout với sidebar
│   │   │   ├── Layout.css
│   │   │   ├── StatusBadge.js         # Status indicator component
│   │   │   ├── StatusBadge.css
│   │   │   └── ResourceIcon.js        # Icon cho resource types
│   │   ├── stack-detail/
│   │   │   ├── StackOverviewTab.js    # Tab 1: Resource details
│   │   │   ├── StackOverviewTab.css
│   │   │   ├── StackTopologyTab.js    # Tab 2: Topology diagram
│   │   │   ├── StackTopologyTab.css
│   │   │   ├── StackMetricsTab.js     # Tab 3: Charts & metrics
│   │   │   ├── StackMetricsTab.css
│   │   │   ├── StackLogsTab.js        # Tab 4: Unified logs
│   │   │   ├── StackLogsTab.css
│   │   │   ├── StackConfigTab.js      # Tab 5: Configuration
│   │   │   └── StackConfigTab.css
│   │   ├── Login.js
│   │   ├── StackDashboard.js          # Main dashboard with stack cards
│   │   ├── StackDashboard.css
│   │   ├── StackDetailPage.js         # Stack detail với 5 tabs
│   │   └── StackDetailPage.css
│   ├── api.js                          # API client (auth, cluster, nginx, docker, stack)
│   ├── App.js
│   ├── App.css
│   ├── index.js
│   └── index.css
├── package.json
└── README.md
```

## 🚀 Cài đặt và Chạy

### 1. Cài đặt dependencies

```powershell
cd d:\iaas\vcs-frontend
npm install
```

### 2. Chạy development server

```powershell
npm start
```

Frontend sẽ chạy tại: `http://localhost:3000`

### 3. Build production

```powershell
npm run build
```

## 🔑 Authentication

### Default credentials
- **Username**: `admin`
- **Password**: `password123`

Backend auth service: `http://localhost:8082`

## 📊 API Endpoints

Frontend kết nối với các service sau:

### Authentication Service (Port 8082)
- `POST /auth/login` - Đăng nhập
- `POST /auth/refresh` - Refresh token

### Provisioning Service (Port 8083)
- `GET /api/v1/stacks` - Lấy danh sách stacks
- `GET /api/v1/stacks/:id` - Chi tiết stack
- `POST /api/v1/stacks` - Tạo stack mới
- `DELETE /api/v1/stacks/:id` - Xóa stack
- `POST /api/v1/stacks/:id/start` - Start stack
- `POST /api/v1/stacks/:id/stop` - Stop stack
- `POST /api/v1/stacks/:id/restart` - Restart stack
- `GET /api/v1/stacks/:id/logs` - Lấy logs
- `GET /api/v1/stacks/:id/metrics` - Lấy metrics

## 🎨 Design System

### Colors
```css
/* Status Colors */
--color-success: #10B981  (Green)
--color-warning: #F59E0B  (Yellow)
--color-error: #EF4444    (Red)
--color-info: #3B82F6     (Blue)

/* Background */
--bg-primary: #F9FAFB
--bg-secondary: #FFFFFF
--bg-dark: #1F2937
--bg-accent: #3B82F6

/* Text */
--text-primary: #111827
--text-secondary: #6B7280
--text-muted: #9CA3AF
```

### Typography
- **Font Family**: Inter, system fonts
- **Heading Sizes**: 32px, 24px, 18px, 16px
- **Body**: 14px (normal), 12px (small)
- **Code**: JetBrains Mono, Courier New

## 🧩 Components Overview

### 1. StackDashboard
- System overview với 8 stat cards
- Search & filters (Environment, Status)
- Stack cards grid
- Quick actions (View Details, Logs, More)

### 2. StackDetailPage
**Header:**
- Stack name, status badge
- Environment, tags, metadata
- Action buttons (Restart All, Stop All, Settings)

**Tabs:**

**a) Overview Tab**
- Stack health summary (Overall health %, Active components, Resource usage)
- Resource cards (PostgreSQL, Nginx, Docker) với:
  - Status badge
  - Role, Infrastructure ID
  - Dependencies
  - Outputs (endpoints, connection strings)

**b) Topology Tab**
- Interactive topology diagram
- Resources grouped by role (Gateway → App → Cache → Database)
- Visual connections & data flow
- Dependencies list

**c) Metrics Tab**
- Aggregated resource usage (CPU, Memory charts)
- Database performance (QPS, Latency, Connections)
- Nginx traffic (Requests/sec, Success rate, Error rate)
- Time range selector (1h, 6h, 24h, 7d, 30d)
- Auto-refresh (30s)

**d) Logs Tab**
- Unified logs từ all resources
- Filters (Resource, Level, Search)
- Color-coded by level (ERROR=red, WARN=yellow, INFO=blue, DEBUG=purple)
- Auto-refresh (5s)
- Download logs

**e) Configuration Tab**
- Basic information (editable)
- Resource dependencies & creation order
- Stack operations (Restart, Stop, Start, Clone, Export, Delete)
- Metadata (Created, Updated, Total resources, Status)

### 3. Common Components

**Layout:**
- Header với user menu & notifications
- Sidebar navigation (Overview, Stacks, Deploy, Metrics, Alerts, Settings)
- Main content area

**StatusBadge:**
- Green: Running/Healthy/Active
- Yellow: Warning/Degraded
- Red: Failed/Error/Down
- Gray: Stopped/Inactive
- Blue: Starting/Initializing

**ResourceIcon:**
- Database icon cho PostgreSQL
- Globe icon cho Nginx
- Container icon cho Docker
- Layers icon cho Stack

## 🎯 Tính năng chính

### Stack Management
✅ View all stacks với filters
✅ Stack cards với resource count
✅ Quick stats (Total, Running, Issues, Cost)
✅ Search stacks by name
✅ Filter by environment & status

### Stack Detail
✅ 5 tabs navigation
✅ Resource details với outputs
✅ Topology visualization
✅ Real-time metrics charts
✅ Unified logs viewer
✅ Configuration management

### Real-time Updates
✅ Auto-refresh metrics (30s)
✅ Auto-refresh logs (5s)
✅ Toast notifications
✅ WebSocket support (ready)

### Operations
✅ Start/Stop/Restart stack
✅ Clone stack
✅ Export template
✅ Delete stack
✅ Download logs

## 🐛 Troubleshooting

### Port conflicts
```powershell
# Check ports
netstat -ano | findstr "3000"
netstat -ano | findstr "8082"
netstat -ano | findstr "8083"
```

### CORS issues
Backend services cần enable CORS cho `http://localhost:3000`

### API connection errors
1. Kiểm tra auth service: `http://localhost:8082/health`
2. Kiểm tra provisioning service: `http://localhost:8083/api/v1/health`
3. Login lại để refresh token

### Build errors
```powershell
# Clear node_modules & reinstall
rm -rf node_modules package-lock.json
npm install
```

## 📸 Screenshots

### Dashboard Overview
- System stats (8 cards)
- Stack cards grid
- Search & filters

### Stack Detail - Overview Tab
- Health summary
- PostgreSQL cluster với replication status
- Nginx gateway với domains & routes
- Docker services với environment variables

### Stack Detail - Topology Tab
- Visual topology diagram
- Resources grouped by layers
- Dependencies graph

### Stack Detail - Metrics Tab
- CPU/Memory charts
- Database QPS & Latency
- Nginx traffic & errors

### Stack Detail - Logs Tab
- Real-time logs stream
- Color-coded by level
- Search & filter

## 🔮 Future Enhancements

- [ ] Create Stack Wizard (4-step)
- [ ] Stack Templates library
- [ ] Dark mode
- [ ] WebSocket real-time updates
- [ ] Advanced metrics (Prometheus)
- [ ] Alert management
- [ ] Cost analytics
- [ ] Backup/Restore UI
- [ ] Drag-and-drop topology editor
- [ ] Multi-tenant support

## 📚 Tài liệu tham khảo

- [React Documentation](https://react.dev/)
- [Recharts Documentation](https://recharts.org/)
- [Lucide Icons](https://lucide.dev/)
- [React Hot Toast](https://react-hot-toast.com/)

---

**Version**: 1.0.0  
**Last Updated**: November 24, 2025  
**Author**: VCS Infrastructure Team
