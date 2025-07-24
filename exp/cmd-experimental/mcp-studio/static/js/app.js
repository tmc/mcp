// MCP Studio - Main Application JavaScript

class MCPStudio {
    constructor() {
        this.ws = null;
        this.connected = false;
        this.projectId = null;
        this.reconnectInterval = null;
        this.heartbeatInterval = null;
        
        this.init();
    }
    
    init() {
        this.setupWebSocket();
        this.setupEventListeners();
        this.setupModals();
    }
    
    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            this.connected = true;
            this.showNotification('Connected to MCP Studio', 'success');
            
            // Clear reconnect interval
            if (this.reconnectInterval) {
                clearInterval(this.reconnectInterval);
                this.reconnectInterval = null;
            }
            
            // Start heartbeat
            this.startHeartbeat();
        };
        
        this.ws.onclose = () => {
            this.connected = false;
            this.showNotification('Disconnected from MCP Studio', 'error');
            
            // Stop heartbeat
            if (this.heartbeatInterval) {
                clearInterval(this.heartbeatInterval);
                this.heartbeatInterval = null;
            }
            
            // Try to reconnect
            this.reconnect();
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showNotification('Connection error', 'error');
        };
        
        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (error) {
                console.error('Error parsing message:', error);
            }
        };
    }
    
    reconnect() {
        if (this.reconnectInterval) return;
        
        this.reconnectInterval = setInterval(() => {
            if (this.ws.readyState === WebSocket.CLOSED) {
                this.setupWebSocket();
            }
        }, 5000);
    }
    
    startHeartbeat() {
        this.heartbeatInterval = setInterval(() => {
            if (this.connected) {
                this.sendMessage({
                    type: 'ping',
                    timestamp: new Date().toISOString()
                });
            }
        }, 30000);
    }
    
    sendMessage(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }
    
    handleMessage(message) {
        switch (message.type) {
            case 'pong':
                // Heartbeat response
                break;
                
            case 'connected':
                console.log('Connected with ID:', message.id);
                break;
                
            case 'flow_update':
                this.handleFlowUpdate(message.data);
                break;
                
            case 'node_update':
                this.handleNodeUpdate(message.data);
                break;
                
            case 'server_status':
                this.handleServerStatus(message.data);
                break;
                
            case 'error':
                this.showNotification(message.error, 'error');
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }
    
    handleFlowUpdate(data) {
        // Handle flow updates
        if (window.flowEditor) {
            window.flowEditor.updateFlow(data);
        }
    }
    
    handleNodeUpdate(data) {
        // Handle node updates
        if (window.flowEditor) {
            window.flowEditor.updateNode(data);
        }
    }
    
    handleServerStatus(data) {
        // Update server status in UI
        const serverElements = document.querySelectorAll(`[data-server-id="${data.id}"]`);
        serverElements.forEach(element => {
            element.classList.toggle('connected', data.connected);
            element.classList.toggle('disconnected', !data.connected);
        });
    }
    
    setupEventListeners() {
        // Handle page navigation
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-action]')) {
                const action = e.target.dataset.action;
                this.handleAction(action, e.target);
            }
        });
        
        // Handle form submissions
        document.addEventListener('submit', (e) => {
            if (e.target.matches('[data-form]')) {
                e.preventDefault();
                const formType = e.target.dataset.form;
                this.handleFormSubmit(formType, e.target);
            }
        });
        
        // Handle keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
                    case 's':
                        e.preventDefault();
                        this.saveCurrentFlow();
                        break;
                    case 'r':
                        e.preventDefault();
                        this.runCurrentFlow();
                        break;
                }
            }
        });
    }
    
    setupModals() {
        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeModal(e.target.id);
            }
        });
        
        // Close modals on escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const openModal = document.querySelector('.modal.show');
                if (openModal) {
                    this.closeModal(openModal.id);
                }
            }
        });
    }
    
    showModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.add('show');
            document.body.style.overflow = 'hidden';
        }
    }
    
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('show');
            document.body.style.overflow = '';
        }
    }
    
    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        // Add to page
        let container = document.querySelector('.notification-container');
        if (!container) {
            container = document.createElement('div');
            container.className = 'notification-container';
            document.body.appendChild(container);
        }
        
        container.appendChild(notification);
        
        // Show notification
        setTimeout(() => {
            notification.classList.add('show');
        }, 100);
        
        // Auto-hide after 5 seconds
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 5000);
    }
    
    async apiCall(method, path, data = null) {
        const options = {
            method: method,
            headers: {
                'Content-Type': 'application/json',
            },
        };
        
        if (data) {
            options.body = JSON.stringify(data);
        }
        
        try {
            const response = await fetch(`/api${path}`, options);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            this.showNotification(`API Error: ${error.message}`, 'error');
            throw error;
        }
    }
    
    // API methods
    async getProjects() {
        return this.apiCall('GET', '/projects');
    }
    
    async createProject(projectData) {
        return this.apiCall('POST', '/projects', projectData);
    }
    
    async getProject(projectId) {
        return this.apiCall('GET', `/projects/${projectId}`);
    }
    
    async updateProject(projectId, projectData) {
        return this.apiCall('PUT', `/projects/${projectId}`, projectData);
    }
    
    async deleteProject(projectId) {
        return this.apiCall('DELETE', `/projects/${projectId}`);
    }
    
    async getFlows(projectId) {
        return this.apiCall('GET', `/projects/${projectId}/flows`);
    }
    
    async createFlow(projectId, flowData) {
        return this.apiCall('POST', `/projects/${projectId}/flows`, flowData);
    }
    
    async updateFlow(projectId, flowId, flowData) {
        return this.apiCall('PUT', `/projects/${projectId}/flows/${flowId}`, flowData);
    }
    
    async runFlow(projectId, flowId, inputData = {}) {
        return this.apiCall('POST', `/projects/${projectId}/flows/${flowId}/run`, inputData);
    }
    
    async getServers() {
        return this.apiCall('GET', '/servers');
    }
    
    async createServer(serverData) {
        return this.apiCall('POST', '/servers', serverData);
    }
    
    async connectServer(serverId) {
        return this.apiCall('POST', `/servers/${serverId}/connect`);
    }
    
    async disconnectServer(serverId) {
        return this.apiCall('POST', `/servers/${serverId}/disconnect`);
    }
    
    async pingServer(serverId) {
        return this.apiCall('POST', `/servers/${serverId}/ping`);
    }
    
    // Utility methods
    subscribeToProject(projectId) {
        this.projectId = projectId;
        this.sendMessage({
            type: 'subscribe_project',
            data: projectId
        });
    }
    
    saveCurrentFlow() {
        if (window.flowEditor) {
            window.flowEditor.save();
        }
    }
    
    runCurrentFlow() {
        if (window.flowEditor) {
            window.flowEditor.run();
        }
    }
    
    handleAction(action, element) {
        switch (action) {
            case 'create-project':
                this.showModal('createProjectModal');
                break;
            case 'edit-project':
                const projectId = element.dataset.projectId;
                this.editProject(projectId);
                break;
            case 'delete-project':
                const deleteProjectId = element.dataset.projectId;
                this.deleteProjectConfirm(deleteProjectId);
                break;
            case 'connect-server':
                const serverId = element.dataset.serverId;
                this.connectServer(serverId);
                break;
            case 'disconnect-server':
                const disconnectServerId = element.dataset.serverId;
                this.disconnectServer(disconnectServerId);
                break;
        }
    }
    
    handleFormSubmit(formType, form) {
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        
        switch (formType) {
            case 'create-project':
                this.createProject(data);
                break;
            case 'edit-project':
                const projectId = form.dataset.projectId;
                this.updateProject(projectId, data);
                break;
            case 'create-server':
                this.createServer(data);
                break;
        }
    }
    
    async editProject(projectId) {
        try {
            const project = await this.getProject(projectId);
            // Show edit modal with project data
            this.showModal('editProjectModal');
            // Populate form with project data
            const form = document.getElementById('editProjectForm');
            if (form) {
                form.elements.name.value = project.name;
                form.elements.description.value = project.description;
                form.dataset.projectId = projectId;
            }
        } catch (error) {
            console.error('Error loading project:', error);
        }
    }
    
    async deleteProjectConfirm(projectId) {
        if (confirm('Are you sure you want to delete this project?')) {
            try {
                await this.deleteProject(projectId);
                this.showNotification('Project deleted successfully', 'success');
                // Refresh the page or remove the project from UI
                location.reload();
            } catch (error) {
                console.error('Error deleting project:', error);
            }
        }
    }
}

// Initialize the application
window.mcpStudio = new MCPStudio();

// Global functions for backward compatibility
window.showModal = (modalId) => window.mcpStudio.showModal(modalId);
window.closeModal = (modalId) => window.mcpStudio.closeModal(modalId);
window.showNotification = (message, type) => window.mcpStudio.showNotification(message, type);