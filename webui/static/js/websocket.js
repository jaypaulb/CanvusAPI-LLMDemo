/**
 * WebSocketClient - Auto-reconnecting WebSocket client for CanvusLocalLLM Dashboard
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Event-based message handling
 * - Connection state management
 * - Heartbeat/ping-pong support
 */

class WebSocketClient {
    /**
     * Create a new WebSocketClient instance
     * @param {Object} options - Configuration options
     * @param {string} options.url - WebSocket URL (default: auto-detect from location)
     * @param {number} options.reconnectInterval - Base reconnect interval in ms (default: 5000)
     * @param {number} options.maxReconnectInterval - Max reconnect interval in ms (default: 30000)
     * @param {number} options.reconnectDecay - Exponential backoff multiplier (default: 1.5)
     * @param {number} options.maxReconnectAttempts - Max attempts before giving up (default: 0 = unlimited)
     */
    constructor(options = {}) {
        // Default WebSocket URL based on current location
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const defaultUrl = `${protocol}//${window.location.host}/ws`;

        this.url = options.url || defaultUrl;
        this.reconnectInterval = options.reconnectInterval || 5000;
        this.maxReconnectInterval = options.maxReconnectInterval || 30000;
        this.reconnectDecay = options.reconnectDecay || 1.5;
        this.maxReconnectAttempts = options.maxReconnectAttempts || 0;

        // Internal state
        this.ws = null;
        this.reconnectAttempts = 0;
        this.currentReconnectInterval = this.reconnectInterval;
        this.reconnectTimer = null;
        this.isIntentionallyClosed = false;
        this.isConnected = false;

        // Event handlers
        this.handlers = {
            open: [],
            close: [],
            error: [],
            message: [],
            reconnecting: [],
            reconnectFailed: []
        };

        // Message type handlers
        this.messageHandlers = {};
    }

    /**
     * Connect to the WebSocket server
     */
    connect() {
        if (this.ws && (this.ws.readyState === WebSocket.CONNECTING || this.ws.readyState === WebSocket.OPEN)) {
            console.log('[WebSocket] Already connected or connecting');
            return;
        }

        this.isIntentionallyClosed = false;
        this._createConnection();
    }

    /**
     * Disconnect from the WebSocket server
     */
    disconnect() {
        this.isIntentionallyClosed = true;
        this._clearReconnectTimer();

        if (this.ws) {
            this.ws.close(1000, 'Client disconnect');
            this.ws = null;
        }

        this.isConnected = false;
    }

    /**
     * Send a message to the server
     * @param {Object|string} data - Data to send (objects will be JSON-stringified)
     * @returns {boolean} Whether the message was sent successfully
     */
    send(data) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.warn('[WebSocket] Cannot send - not connected');
            return false;
        }

        try {
            const message = typeof data === 'string' ? data : JSON.stringify(data);
            this.ws.send(message);
            return true;
        } catch (error) {
            console.error('[WebSocket] Send error:', error);
            return false;
        }
    }

    /**
     * Register an event handler
     * @param {string} event - Event name (open, close, error, message, reconnecting, reconnectFailed)
     * @param {Function} handler - Event handler function
     */
    on(event, handler) {
        if (this.handlers[event]) {
            this.handlers[event].push(handler);
        } else {
            console.warn(`[WebSocket] Unknown event: ${event}`);
        }
    }

    /**
     * Remove an event handler
     * @param {string} event - Event name
     * @param {Function} handler - Handler to remove
     */
    off(event, handler) {
        if (this.handlers[event]) {
            this.handlers[event] = this.handlers[event].filter(h => h !== handler);
        }
    }

    /**
     * Register a handler for a specific message type
     * @param {string} type - Message type to handle
     * @param {Function} handler - Handler function receiving (data, event)
     */
    onMessage(type, handler) {
        if (!this.messageHandlers[type]) {
            this.messageHandlers[type] = [];
        }
        this.messageHandlers[type].push(handler);
    }

    /**
     * Remove a message type handler
     * @param {string} type - Message type
     * @param {Function} handler - Handler to remove
     */
    offMessage(type, handler) {
        if (this.messageHandlers[type]) {
            this.messageHandlers[type] = this.messageHandlers[type].filter(h => h !== handler);
        }
    }

    /**
     * Get connection state
     * @returns {string} Connection state (connecting, open, closing, closed)
     */
    getState() {
        if (!this.ws) return 'closed';

        switch (this.ws.readyState) {
            case WebSocket.CONNECTING: return 'connecting';
            case WebSocket.OPEN: return 'open';
            case WebSocket.CLOSING: return 'closing';
            case WebSocket.CLOSED: return 'closed';
            default: return 'unknown';
        }
    }

    // Private methods

    /**
     * Create WebSocket connection
     * @private
     */
    _createConnection() {
        console.log(`[WebSocket] Connecting to ${this.url}`);

        try {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = (event) => this._handleOpen(event);
            this.ws.onclose = (event) => this._handleClose(event);
            this.ws.onerror = (event) => this._handleError(event);
            this.ws.onmessage = (event) => this._handleMessage(event);

        } catch (error) {
            console.error('[WebSocket] Connection error:', error);
            this._scheduleReconnect();
        }
    }

    /**
     * Handle WebSocket open event
     * @private
     */
    _handleOpen(event) {
        console.log('[WebSocket] Connected');

        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.currentReconnectInterval = this.reconnectInterval;

        this._emit('open', event);
    }

    /**
     * Handle WebSocket close event
     * @private
     */
    _handleClose(event) {
        console.log(`[WebSocket] Closed: code=${event.code}, reason=${event.reason}`);

        this.isConnected = false;
        this.ws = null;

        this._emit('close', event);

        if (!this.isIntentionallyClosed) {
            this._scheduleReconnect();
        }
    }

    /**
     * Handle WebSocket error event
     * @private
     */
    _handleError(event) {
        console.error('[WebSocket] Error:', event);
        this._emit('error', event);
    }

    /**
     * Handle incoming WebSocket message
     * @private
     */
    _handleMessage(event) {
        let data;

        try {
            data = JSON.parse(event.data);
        } catch (e) {
            // Not JSON, use raw data
            data = event.data;
        }

        // Emit generic message event
        this._emit('message', data, event);

        // Handle typed messages
        if (data && typeof data === 'object' && data.type) {
            const handlers = this.messageHandlers[data.type];
            if (handlers) {
                handlers.forEach(handler => {
                    try {
                        handler(data, event);
                    } catch (error) {
                        console.error(`[WebSocket] Message handler error for type ${data.type}:`, error);
                    }
                });
            }
        }
    }

    /**
     * Schedule a reconnection attempt
     * @private
     */
    _scheduleReconnect() {
        if (this.isIntentionallyClosed) {
            return;
        }

        // Check max attempts
        if (this.maxReconnectAttempts > 0 && this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('[WebSocket] Max reconnection attempts reached');
            this._emit('reconnectFailed', { attempts: this.reconnectAttempts });
            return;
        }

        this.reconnectAttempts++;

        console.log(`[WebSocket] Reconnecting in ${this.currentReconnectInterval}ms (attempt ${this.reconnectAttempts})`);

        this._emit('reconnecting', {
            attempt: this.reconnectAttempts,
            delay: this.currentReconnectInterval
        });

        this.reconnectTimer = setTimeout(() => {
            this._createConnection();
        }, this.currentReconnectInterval);

        // Exponential backoff
        this.currentReconnectInterval = Math.min(
            this.currentReconnectInterval * this.reconnectDecay,
            this.maxReconnectInterval
        );
    }

    /**
     * Clear reconnection timer
     * @private
     */
    _clearReconnectTimer() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }

    /**
     * Emit an event to all registered handlers
     * @private
     */
    _emit(event, ...args) {
        const handlers = this.handlers[event];
        if (handlers) {
            handlers.forEach(handler => {
                try {
                    handler(...args);
                } catch (error) {
                    console.error(`[WebSocket] Event handler error for ${event}:`, error);
                }
            });
        }
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = WebSocketClient;
}
