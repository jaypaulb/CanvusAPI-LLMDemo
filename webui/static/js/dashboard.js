/**
 * DashboardApp - Main dashboard application for CanvusLocalLLM
 *
 * Handles:
 * - Initial data loading from REST API
 * - WebSocket connection for real-time updates
 * - UI rendering and event handling
 * - GPU metrics visualization
 */

class DashboardApp {
    constructor() {
        // WebSocket client
        this.ws = new WebSocketClient({
            reconnectInterval: 5000,
            maxReconnectInterval: 30000,
            reconnectDecay: 1.5
        });

        // State
        this.status = null;
        this.canvases = [];
        this.tasks = [];
        this.metrics = null;
        this.gpuMetrics = null;
        this.gpuHistory = [];
        this.activityLog = [];
        this.activityFilter = 'all';

        // GPU chart (Chart.js)
        this.gpuChart = null;
        this.gpuChartLabels = [];
        this.gpuChartUtilization = [];
        this.gpuChartMemory = [];
        this.maxChartPoints = 720; // 1 hour at 5-second intervals (60 * 60 / 5 = 720)
        this.gpuAvailable = false;

        // DOM elements (cached after init)
        this.elements = {};

        // Initialize
        this.init();
    }

    /**
     * Initialize the dashboard
     */
    async init() {
        console.log('[Dashboard] Initializing...');

        // Cache DOM elements
        this.cacheElements();

        // Setup event listeners
        this.setupEventListeners();

        // Setup WebSocket handlers
        this.setupWebSocket();

        // Load initial data
        await this.loadInitialData();

        // Connect WebSocket
        this.connectWebSocket();

        // Start time update
        this.startTimeUpdate();

        console.log('[Dashboard] Initialized');
    }

    /**
     * Cache DOM element references
     */
    cacheElements() {
        this.elements = {
            // Connection status
            connectionStatus: document.getElementById('connection-status'),
            versionInfo: document.getElementById('version-info'),

            // System status
            systemHealthBadge: document.getElementById('system-health-badge'),
            systemHealth: document.getElementById('system-health'),
            systemUptime: document.getElementById('system-uptime'),
            gpuAvailable: document.getElementById('gpu-available'),
            lastCheck: document.getElementById('last-check'),

            // GPU metrics
            gpuStatusBadge: document.getElementById('gpu-status-badge'),
            gpuName: document.getElementById('gpu-name'),
            gpuDriver: document.getElementById('gpu-driver'),
            gpuUtilization: document.getElementById('gpu-utilization'),
            gpuUtilizationBar: document.getElementById('gpu-utilization-bar'),
            gpuMemory: document.getElementById('gpu-memory'),
            gpuMemoryBar: document.getElementById('gpu-memory-bar'),
            gpuTemperature: document.getElementById('gpu-temperature'),
            gpuTemperatureBar: document.getElementById('gpu-temperature-bar'),
            gpuPower: document.getElementById('gpu-power'),
            gpuPowerBar: document.getElementById('gpu-power-bar'),
            gpuChart: document.getElementById('gpu-chart'),
            gpuChartContainer: document.getElementById('gpu-chart-container'),

            // Canvas status
            canvasCountBadge: document.getElementById('canvas-count-badge'),
            canvasList: document.getElementById('canvas-list'),

            // Processing metrics
            totalProcessed: document.getElementById('total-processed'),
            totalSuccess: document.getElementById('total-success'),
            totalErrors: document.getElementById('total-errors'),
            successRate: document.getElementById('success-rate'),
            metricsByType: document.getElementById('metrics-by-type'),

            // Queue
            queueCountBadge: document.getElementById('queue-count-badge'),
            queueList: document.getElementById('queue-list'),

            // Activity
            activityLog: document.getElementById('activity-log'),
            activityFilter: document.getElementById('activity-filter'),
            clearActivityBtn: document.getElementById('clear-activity-btn'),

            // Footer
            footerStatus: document.getElementById('footer-status'),
            footerVersion: document.getElementById('footer-version'),
            footerTime: document.getElementById('footer-time')
        };
    }

    /**
     * Setup DOM event listeners
     */
    setupEventListeners() {
        // Activity filter
        if (this.elements.activityFilter) {
            this.elements.activityFilter.addEventListener('change', (e) => {
                this.activityFilter = e.target.value;
                this.renderActivityLog();
            });
        }

        // Clear activity button
        if (this.elements.clearActivityBtn) {
            this.elements.clearActivityBtn.addEventListener('click', () => {
                this.activityLog = [];
                this.renderActivityLog();
            });
        }
    }

    /**
     * Setup WebSocket message handlers
     */
    setupWebSocket() {
        // Connection events
        this.ws.on('open', () => {
            this.updateConnectionStatus('connected');
            console.log('[Dashboard] WebSocket connected');
        });

        this.ws.on('close', () => {
            this.updateConnectionStatus('disconnected');
            console.log('[Dashboard] WebSocket disconnected');
        });

        this.ws.on('reconnecting', (info) => {
            this.updateConnectionStatus('connecting');
            console.log(`[Dashboard] Reconnecting (attempt ${info.attempt})...`);
        });

        this.ws.on('error', (error) => {
            console.error('[Dashboard] WebSocket error:', error);
        });

        // Message type handlers
        this.ws.onMessage('status', (data) => this.handleStatusUpdate(data));
        this.ws.onMessage('canvas_status', (data) => this.handleCanvasUpdate(data));
        this.ws.onMessage('task_started', (data) => this.handleTaskStarted(data));
        this.ws.onMessage('task_completed', (data) => this.handleTaskCompleted(data));
        this.ws.onMessage('task_error', (data) => this.handleTaskError(data));
        this.ws.onMessage('metrics', (data) => this.handleMetricsUpdate(data));
        this.ws.onMessage('gpu', (data) => this.handleGPUUpdate(data));
    }

    /**
     * Connect WebSocket
     */
    connectWebSocket() {
        this.updateConnectionStatus('connecting');
        this.ws.connect();
    }

    /**
     * Load initial data from REST API
     */
    async loadInitialData() {
        try {
            // Load all data in parallel
            const [status, canvases, tasks, metrics, gpu] = await Promise.all([
                this.fetchAPI('/api/status'),
                this.fetchAPI('/api/canvases'),
                this.fetchAPI('/api/tasks?limit=50'),
                this.fetchAPI('/api/metrics'),
                this.fetchAPI('/api/gpu?history=60')
            ]);

            // Update state and render
            if (status) {
                this.status = status;
                this.renderStatus();
            }

            if (canvases) {
                this.canvases = canvases.canvases || [];
                this.renderCanvases();
            }

            if (tasks) {
                this.tasks = tasks.tasks || [];
                this.activityLog = this.tasks.slice(0, 100);
                this.renderActivityLog();
            }

            if (metrics) {
                this.metrics = metrics;
                this.renderMetrics();
            }

            if (gpu) {
                this.gpuMetrics = gpu.current;
                this.gpuHistory = gpu.history || [];
                this.renderGPU();
            }

        } catch (error) {
            console.error('[Dashboard] Failed to load initial data:', error);
        }
    }

    /**
     * Fetch data from API
     */
    async fetchAPI(endpoint) {
        try {
            const response = await fetch(endpoint);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`[Dashboard] API fetch failed: ${endpoint}`, error);
            return null;
        }
    }

    // WebSocket message handlers

    handleStatusUpdate(data) {
        this.status = data;
        this.renderStatus();
    }

    handleCanvasUpdate(data) {
        const canvas = data.canvas;
        if (!canvas) return;

        const index = this.canvases.findIndex(c => c.id === canvas.id);
        if (index >= 0) {
            this.canvases[index] = canvas;
        } else {
            this.canvases.push(canvas);
        }
        this.renderCanvases();
    }

    handleTaskStarted(data) {
        // Add to activity log
        this.addActivity({
            ...data.task,
            status: 'processing'
        });

        // Add to queue
        this.renderQueue();
    }

    handleTaskCompleted(data) {
        // Update activity log
        this.updateActivity(data.task);

        // Update metrics
        if (data.metrics) {
            this.metrics = data.metrics;
            this.renderMetrics();
        }
    }

    handleTaskError(data) {
        // Update activity log with error
        this.updateActivity({
            ...data.task,
            status: 'error',
            error: data.error
        });
    }

    handleMetricsUpdate(data) {
        this.metrics = data;
        this.renderMetrics();
    }

    handleGPUUpdate(data) {
        this.gpuMetrics = data.current || data;
        if (data.history) {
            this.gpuHistory = data.history;
        }
        this.renderGPU();
        this.updateGPUChart();
    }

    // Rendering methods

    renderStatus() {
        if (!this.status) return;

        // Health badge
        const healthClass = this.getHealthClass(this.status.health);
        this.setElementText('systemHealthBadge', this.status.health?.toUpperCase() || '--');
        this.setElementClass('systemHealthBadge', `widget-badge badge-${healthClass}`);

        // Health value
        this.setElementText('systemHealth', this.status.health || '--');
        this.setElementClass('systemHealth', `status-value status-${this.status.health?.toLowerCase() || ''}`);

        // Uptime
        this.setElementText('systemUptime', this.status.uptime || '--');

        // GPU available
        this.setElementText('gpuAvailable', this.status.gpu_available ? 'Yes' : 'No');

        // Last check
        if (this.status.last_check) {
            const lastCheck = new Date(this.status.last_check);
            this.setElementText('lastCheck', this.formatTime(lastCheck));
        }

        // Version info
        if (this.status.version) {
            this.setElementText('versionInfo', `v${this.status.version}`);
            this.setElementText('footerVersion', `CanvusLocalLLM v${this.status.version}`);
        }
    }

    renderCanvases() {
        if (!this.elements.canvasList) return;

        this.setElementText('canvasCountBadge', this.canvases.length.toString());

        if (this.canvases.length === 0) {
            this.elements.canvasList.innerHTML = '<div class="empty-state">No canvases connected</div>';
            return;
        }

        const html = this.canvases.map(canvas => {
            const statusClass = canvas.connected ? 'connected' : 'disconnected';
            return `
                <div class="canvas-item">
                    <div class="canvas-name">
                        <span class="canvas-status-dot ${statusClass}"></span>
                        ${this.escapeHtml(canvas.name || canvas.id)}
                    </div>
                    <div class="canvas-widgets">${canvas.widget_count || 0} widgets</div>
                </div>
            `;
        }).join('');

        this.elements.canvasList.innerHTML = html;
    }

    renderMetrics() {
        if (!this.metrics) return;

        this.setElementText('totalProcessed', this.formatNumber(this.metrics.total_processed || 0));
        this.setElementText('totalSuccess', this.formatNumber(this.metrics.total_success || 0));
        this.setElementText('totalErrors', this.formatNumber(this.metrics.total_errors || 0));
        this.setElementText('successRate', `${(this.metrics.success_rate || 0).toFixed(1)}%`);

        // Metrics by type
        if (this.elements.metricsByType && this.metrics.by_type) {
            const html = Object.entries(this.metrics.by_type).map(([type, stats]) => `
                <div class="type-metric">
                    <div class="type-name">${this.formatTaskType(type)}</div>
                    <div class="type-stats">
                        <span>${stats.total_processed || 0} processed</span>
                        <span>${stats.total_errors || 0} errors</span>
                    </div>
                </div>
            `).join('');

            this.elements.metricsByType.innerHTML = html || '<div class="empty-state">No type metrics</div>';
        }
    }

    renderGPU() {
        if (!this.gpuMetrics) {
            this.gpuAvailable = false;
            this.setElementText('gpuStatusBadge', 'N/A');
            this.setElementClass('gpuStatusBadge', 'widget-badge');
            this.showGPUChartUnavailable('No GPU detected');
            return;
        }

        // Mark GPU as available
        this.gpuAvailable = true;

        // Status badge
        this.setElementText('gpuStatusBadge', 'Active');
        this.setElementClass('gpuStatusBadge', 'widget-badge badge-success');

        // GPU info
        this.setElementText('gpuName', this.gpuMetrics.name || 'GPU');
        this.setElementText('gpuDriver', this.gpuMetrics.driver_version || '--');

        // Utilization
        const utilization = this.gpuMetrics.utilization || 0;
        this.setElementText('gpuUtilization', `${utilization}%`);
        this.setBarWidth('gpuUtilizationBar', utilization);

        // Memory
        const memUsed = this.gpuMetrics.memory_used || 0;
        const memTotal = this.gpuMetrics.memory_total || 1;
        const memPercent = (memUsed / memTotal) * 100;
        this.setElementText('gpuMemory', `${memUsed} / ${memTotal} MB`);
        this.setBarWidth('gpuMemoryBar', memPercent);

        // Temperature
        const temp = this.gpuMetrics.temperature || 0;
        this.setElementText('gpuTemperature', `${temp}°C`);
        this.setBarWidth('gpuTemperatureBar', Math.min(temp, 100)); // Clamp to 100

        // Power
        const powerUsed = this.gpuMetrics.power_draw || 0;
        const powerLimit = this.gpuMetrics.power_limit || 1;
        const powerPercent = (powerUsed / powerLimit) * 100;
        this.setElementText('gpuPower', `${powerUsed.toFixed(0)} / ${powerLimit.toFixed(0)} W`);
        this.setBarWidth('gpuPowerBar', powerPercent);

        // Load historical GPU data into chart on initial render
        if (this.gpuHistory && this.gpuHistory.length > 0 && !this.gpuChart) {
            this.loadGPUHistory();
        }
    }

    /**
     * Initialize the Chart.js GPU chart
     */
    initGPUChart() {
        const canvas = this.elements.gpuChart;
        if (!canvas) return;

        // Check if Chart.js is available
        if (typeof Chart === 'undefined') {
            console.warn('[Dashboard] Chart.js not loaded, GPU chart unavailable');
            this.showGPUChartUnavailable('Chart.js not loaded');
            return;
        }

        const ctx = canvas.getContext('2d');

        this.gpuChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: this.gpuChartLabels,
                datasets: [
                    {
                        label: 'Utilization %',
                        data: this.gpuChartUtilization,
                        borderColor: '#58a6ff',
                        backgroundColor: 'rgba(88, 166, 255, 0.1)',
                        borderWidth: 2,
                        fill: true,
                        tension: 0.3,
                        pointRadius: 0,
                        pointHoverRadius: 4
                    },
                    {
                        label: 'Memory %',
                        data: this.gpuChartMemory,
                        borderColor: '#3fb950',
                        backgroundColor: 'rgba(63, 185, 80, 0.1)',
                        borderWidth: 2,
                        fill: true,
                        tension: 0.3,
                        pointRadius: 0,
                        pointHoverRadius: 4
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: {
                    duration: 0 // Disable animation for real-time updates
                },
                interaction: {
                    intersect: false,
                    mode: 'index'
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top',
                        align: 'start',
                        labels: {
                            color: '#8b949e',
                            usePointStyle: true,
                            pointStyle: 'line',
                            boxWidth: 30,
                            padding: 15,
                            font: {
                                size: 11,
                                family: '-apple-system, BlinkMacSystemFont, sans-serif'
                            }
                        }
                    },
                    tooltip: {
                        backgroundColor: '#21262d',
                        titleColor: '#e6edf3',
                        bodyColor: '#8b949e',
                        borderColor: '#30363d',
                        borderWidth: 1,
                        padding: 10,
                        displayColors: true,
                        callbacks: {
                            title: (items) => {
                                if (items.length > 0) {
                                    return items[0].label || '';
                                }
                                return '';
                            },
                            label: (context) => {
                                return `${context.dataset.label}: ${context.parsed.y.toFixed(1)}%`;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            color: '#21262d',
                            lineWidth: 1
                        },
                        ticks: {
                            color: '#6e7681',
                            font: {
                                size: 10
                            },
                            maxRotation: 0,
                            autoSkip: true,
                            maxTicksLimit: 8
                        }
                    },
                    y: {
                        display: true,
                        min: 0,
                        max: 100,
                        grid: {
                            color: '#21262d',
                            lineWidth: 1
                        },
                        ticks: {
                            color: '#6e7681',
                            font: {
                                size: 10
                            },
                            stepSize: 25,
                            callback: (value) => `${value}%`
                        }
                    }
                }
            }
        });
    }

    /**
     * Show N/A placeholder when GPU is unavailable
     */
    showGPUChartUnavailable(reason = 'GPU not available') {
        const container = this.elements.gpuChartContainer;
        if (!container) return;

        container.innerHTML = `
            <div class="gpu-chart-unavailable">
                <span class="gpu-chart-na">N/A</span>
                <span class="gpu-chart-reason">${this.escapeHtml(reason)}</span>
            </div>
        `;
    }

    /**
     * Update GPU chart with new data point
     */
    updateGPUChart() {
        // Check if GPU is available
        if (!this.gpuAvailable || !this.gpuMetrics) {
            if (this.gpuChart) {
                this.gpuChart.destroy();
                this.gpuChart = null;
            }
            this.showGPUChartUnavailable('No GPU data available');
            return;
        }

        // Initialize chart if needed
        if (!this.gpuChart && this.elements.gpuChart) {
            // Restore canvas if it was replaced with N/A message
            const container = this.elements.gpuChartContainer;
            if (container && !container.querySelector('#gpu-chart')) {
                container.innerHTML = '<canvas id="gpu-chart" width="400" height="120"></canvas>';
                this.elements.gpuChart = document.getElementById('gpu-chart');
            }
            this.initGPUChart();
        }

        if (!this.gpuChart) return;

        // Add new data point
        const now = new Date();
        const timeLabel = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

        this.gpuChartLabels.push(timeLabel);
        this.gpuChartUtilization.push(this.gpuMetrics.utilization || 0);

        // Calculate memory percentage
        const memPercent = this.gpuMetrics.memory_used && this.gpuMetrics.memory_total
            ? (this.gpuMetrics.memory_used / this.gpuMetrics.memory_total) * 100
            : 0;
        this.gpuChartMemory.push(memPercent);

        // Keep only last N points (1 hour at 5-second intervals)
        while (this.gpuChartLabels.length > this.maxChartPoints) {
            this.gpuChartLabels.shift();
            this.gpuChartUtilization.shift();
            this.gpuChartMemory.shift();
        }

        // Update chart
        this.gpuChart.update('none'); // 'none' mode for no animation
    }

    /**
     * Load initial GPU history into chart
     */
    loadGPUHistory() {
        if (!this.gpuHistory || this.gpuHistory.length === 0) return;

        // Clear existing data
        this.gpuChartLabels = [];
        this.gpuChartUtilization = [];
        this.gpuChartMemory = [];

        // Load history (most recent last)
        this.gpuHistory.forEach(point => {
            const time = new Date(point.timestamp || point.time);
            const timeLabel = time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

            this.gpuChartLabels.push(timeLabel);
            this.gpuChartUtilization.push(point.utilization || 0);

            const memPercent = point.memory_used && point.memory_total
                ? (point.memory_used / point.memory_total) * 100
                : 0;
            this.gpuChartMemory.push(memPercent);
        });

        // Initialize chart with history
        if (this.gpuAvailable && !this.gpuChart && this.elements.gpuChart) {
            this.initGPUChart();
        } else if (this.gpuChart) {
            this.gpuChart.update('none');
        }
    }

    renderQueue() {
        if (!this.elements.queueList) return;

        const queuedTasks = this.activityLog.filter(t => t.status === 'processing');
        this.setElementText('queueCountBadge', queuedTasks.length.toString());

        if (queuedTasks.length === 0) {
            this.elements.queueList.innerHTML = '<div class="empty-state">No tasks in queue</div>';
            return;
        }

        const html = queuedTasks.map(task => `
            <div class="queue-item processing">
                <span class="queue-type">${this.formatTaskType(task.type)}</span>
                <span class="queue-prompt">${this.escapeHtml(task.prompt || task.details || '--')}</span>
                <span class="queue-time">${this.formatDuration(task.duration)}</span>
            </div>
        `).join('');

        this.elements.queueList.innerHTML = html;
    }

    renderActivityLog() {
        if (!this.elements.activityLog) return;

        // Filter activities
        let filtered = this.activityLog;
        if (this.activityFilter !== 'all') {
            filtered = this.activityLog.filter(a => this.matchActivityType(a.type, this.activityFilter));
        }

        if (filtered.length === 0) {
            this.elements.activityLog.innerHTML = `
                <tr class="empty-row">
                    <td colspan="6" class="empty-state">No recent activity</td>
                </tr>
            `;
            return;
        }

        const html = filtered.slice(0, 100).map(activity => {
            const typeClass = this.getActivityTypeClass(activity.type);
            const statusClass = this.getStatusClass(activity.status);

            return `
                <tr>
                    <td class="col-time">${this.formatTime(new Date(activity.started_at || activity.time))}</td>
                    <td class="col-type">
                        <span class="activity-type type-${typeClass}">${this.formatTaskType(activity.type)}</span>
                    </td>
                    <td class="col-canvas">${this.escapeHtml(activity.canvas_name || '--')}</td>
                    <td class="col-status">
                        <span class="activity-status status-${statusClass}">${activity.status || '--'}</span>
                    </td>
                    <td class="col-duration">
                        <span class="activity-duration">${this.formatDuration(activity.duration)}</span>
                    </td>
                    <td class="col-details">
                        <span class="activity-details" title="${this.escapeHtml(activity.details || activity.error || '')}">${this.escapeHtml(this.truncate(activity.details || activity.error || '--', 50))}</span>
                    </td>
                </tr>
            `;
        }).join('');

        this.elements.activityLog.innerHTML = html;
    }

    // Activity management

    addActivity(activity) {
        // Add to beginning of log
        this.activityLog.unshift(activity);

        // Limit size
        if (this.activityLog.length > 500) {
            this.activityLog = this.activityLog.slice(0, 500);
        }

        this.renderActivityLog();
        this.renderQueue();
    }

    updateActivity(activity) {
        // Find and update existing activity
        const index = this.activityLog.findIndex(a => a.id === activity.id);
        if (index >= 0) {
            this.activityLog[index] = { ...this.activityLog[index], ...activity };
        } else {
            this.activityLog.unshift(activity);
        }

        this.renderActivityLog();
        this.renderQueue();
    }

    // Connection status

    updateConnectionStatus(status) {
        if (!this.elements.connectionStatus) return;

        const statusText = this.elements.connectionStatus.querySelector('.status-text');

        this.elements.connectionStatus.className = `status-indicator status-${status}`;
        if (statusText) {
            statusText.textContent = status === 'connected' ? 'Connected' :
                                     status === 'connecting' ? 'Connecting...' : 'Disconnected';
        }

        this.setElementText('footerStatus', status === 'connected' ? 'Connected' :
                                            status === 'connecting' ? 'Reconnecting...' : 'Disconnected');
    }

    // Time update

    startTimeUpdate() {
        const updateTime = () => {
            if (this.elements.footerTime) {
                this.elements.footerTime.textContent = new Date().toLocaleTimeString();
            }
        };

        updateTime();
        setInterval(updateTime, 1000);
    }

    // Utility methods

    setElementText(key, text) {
        const el = this.elements[key];
        if (el) el.textContent = text;
    }

    setElementClass(key, className) {
        const el = this.elements[key];
        if (el) el.className = className;
    }

    setBarWidth(key, percent) {
        const el = this.elements[key];
        if (el) el.style.width = `${Math.min(100, Math.max(0, percent))}%`;
    }

    formatNumber(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    }

    formatTime(date) {
        if (!date || !(date instanceof Date)) return '--';
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    }

    formatDuration(ms) {
        if (!ms || ms < 0) return '--';
        if (ms < 1000) return `${ms}ms`;
        if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
        return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
    }

    formatTaskType(type) {
        if (!type) return '--';
        const types = {
            'ai_prompt': 'AI',
            'pdf_analysis': 'PDF',
            'ocr': 'OCR',
            'image_gen': 'Image',
            'canvas_analysis': 'Canvas'
        };
        return types[type] || type.charAt(0).toUpperCase() + type.slice(1);
    }

    getHealthClass(health) {
        const map = { healthy: 'success', degraded: 'warning', unhealthy: 'error' };
        return map[health?.toLowerCase()] || '';
    }

    getStatusClass(status) {
        const map = { success: 'success', completed: 'success', error: 'error', failed: 'error', processing: 'processing' };
        return map[status?.toLowerCase()] || '';
    }

    getActivityTypeClass(type) {
        const map = { ai_prompt: 'ai', pdf_analysis: 'pdf', ocr: 'ocr', image_gen: 'image', canvas_analysis: 'canvas' };
        return map[type] || 'ai';
    }

    matchActivityType(taskType, filter) {
        const filterMap = {
            ai: ['ai_prompt'],
            pdf: ['pdf_analysis'],
            ocr: ['ocr'],
            image: ['image_gen'],
            canvas: ['canvas_analysis']
        };
        return filterMap[filter]?.includes(taskType) || false;
    }

    truncate(str, maxLen) {
        if (!str || str.length <= maxLen) return str || '';
        return str.substring(0, maxLen - 3) + '...';
    }

    escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
}

// Initialize dashboard when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new DashboardApp();
});
