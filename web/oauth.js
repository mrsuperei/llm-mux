// OAuth Management UI
// Handles OAuth and Device flow authentication for LLM providers

// Provider configurations
const OAUTH_PROVIDERS = [
    { id: 'claude', name: 'Claude', icon: '🤖', description: 'Anthropic Claude' },
    { id: 'anthropic', name: 'Anthropic', icon: '🧠', description: 'Anthropic API (alias for Claude)' },
    { id: 'codex', name: 'Codex', icon: '💻', description: 'OpenAI Codex' },
    { id: 'gemini', name: 'Gemini', icon: '✨', description: 'Google Gemini' },
    { id: 'gemini-cli', name: 'Gemini CLI', icon: '🔮', description: 'Google Gemini CLI (alias)' },
    { id: 'antigravity', name: 'Antigravity', icon: '🚀', description: 'Antigravity' },
    { id: 'iflow', name: 'iFlow', icon: '🌊', description: 'iFlow' }
];

const DEVICE_FLOW_PROVIDERS = [
    { id: 'qwen', name: 'Qwen', icon: '🎯', description: 'Alibaba Qwen' },
    { id: 'copilot', name: 'Copilot', icon: '✈️', description: 'GitHub Copilot' },
    { id: 'github-copilot', name: 'GitHub Copilot', icon: '🐙', description: 'GitHub Copilot (alias)' }
];

// State management
let currentFlow = null;
let pollInterval = null;

// DOM Elements
const apiUrlInput = document.getElementById('api-url');
const managementKeyInput = document.getElementById('management-key');
const oauthProvidersContainer = document.getElementById('oauth-providers');
const deviceProvidersContainer = document.getElementById('device-providers');
const activeFlowSection = document.getElementById('active-flow');
const flowProviderName = document.getElementById('flow-provider-name');
const flowStatus = document.getElementById('flow-status');
const flowContent = document.getElementById('flow-content');
const cancelFlowBtn = document.getElementById('cancel-flow-btn');
const statusMessages = document.getElementById('status-messages');

// Initialize the UI
function init() {
    renderProviders();
    setupEventListeners();
}

// Render provider buttons
function renderProviders() {
    oauthProvidersContainer.innerHTML = OAUTH_PROVIDERS.map(provider => `
        <button class="provider-card" data-provider="${provider.id}" data-flow-type="oauth">
            <span class="provider-icon">${provider.icon}</span>
            <div class="provider-info">
                <span class="provider-name">${provider.name}</span>
                <span class="provider-desc">${provider.description}</span>
            </div>
            <span class="provider-arrow">→</span>
        </button>
    `).join('');

    deviceProvidersContainer.innerHTML = DEVICE_FLOW_PROVIDERS.map(provider => `
        <button class="provider-card" data-provider="${provider.id}" data-flow-type="device">
            <span class="provider-icon">${provider.icon}</span>
            <div class="provider-info">
                <span class="provider-name">${provider.name}</span>
                <span class="provider-desc">${provider.description}</span>
            </div>
            <span class="provider-arrow">→</span>
        </button>
    `).join('');
}

// Setup event listeners
function setupEventListeners() {
    // Provider button clicks
    document.querySelectorAll('.provider-card').forEach(button => {
        button.addEventListener('click', () => {
            const provider = button.dataset.provider;
            const flowType = button.dataset.flowType;
            startOAuthFlow(provider, flowType);
        });
    });

    // Cancel flow button
    cancelFlowBtn.addEventListener('click', cancelOAuthFlow);

    // API URL change - save to localStorage
    apiUrlInput.addEventListener('change', () => {
        localStorage.setItem('llm-mux-api-url', apiUrlInput.value);
    });

    // Management key change - save to localStorage
    managementKeyInput.addEventListener('change', () => {
        localStorage.setItem('llm_mux_management_key', managementKeyInput.value);
    });

    // Load saved API URL
    const savedApiUrl = localStorage.getItem('llm-mux-api-url');
    if (savedApiUrl) {
        apiUrlInput.value = savedApiUrl;
    }

    // Load saved management key
    const savedManagementKey = localStorage.getItem('llm_mux_management_key');
    if (savedManagementKey) {
        managementKeyInput.value = savedManagementKey;
    }
}

// Get API base URL
function getApiUrl() {
    return apiUrlInput.value.replace(/\/$/, '');
}

// Get management key
function getManagementKey() {
    return managementKeyInput.value;
}

// Show status message
function showMessage(message, type = 'info') {
    const messageEl = document.createElement('div');
    messageEl.className = `message message-${type}`;
    messageEl.innerHTML = `
        <span class="message-icon">${type === 'success' ? '✓' : type === 'error' ? '✗' : 'ℹ'}</span>
        <span class="message-text">${message}</span>
        <button class="message-close" onclick="this.parentElement.remove()">×</button>
    `;
    statusMessages.appendChild(messageEl);

    // Auto-remove after 5 seconds for success messages
    if (type === 'success') {
        setTimeout(() => messageEl.remove(), 5000);
    }
}

// Clear all messages
function clearMessages() {
    statusMessages.innerHTML = '';
}

// Start OAuth flow
async function startOAuthFlow(provider, flowType) {
    const apiUrl = getApiUrl();
    const url = `${apiUrl}/oauth/start`;

    try {
        const headers = {
            'Content-Type': 'application/json'
        };
        
        const managementKey = getManagementKey();
        if (managementKey) {
            headers['X-Management-Key'] = managementKey;
        }

        const response = await fetch(url, {
            method: 'POST',
            headers: headers,
            body: JSON.stringify({ provider })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Failed to start OAuth flow');
        }

        const data = await response.json();

        if (data.status !== 'ok') {
            throw new Error(data.error || 'Failed to start OAuth flow');
        }

        // Store current flow state
        currentFlow = {
            provider,
            flowType: data.flow_type || flowType,
            state: data.state || data.id,
            authUrl: data.auth_url,
            userCode: data.user_code,
            verificationUrl: data.verification_url,
            interval: data.interval || 3
        };

        // Show active flow UI
        showActiveFlow();

        // Handle based on flow type
        if (currentFlow.flowType === 'device') {
            handleDeviceFlow();
        } else {
            handleOAuthFlow();
        }

    } catch (error) {
        showMessage(error.message, 'error');
    }
}

// Show active flow section
function showActiveFlow() {
    const providerInfo = [...OAUTH_PROVIDERS, ...DEVICE_FLOW_PROVIDERS]
        .find(p => p.id === currentFlow.provider);

    flowProviderName.textContent = providerInfo ? providerInfo.name : currentFlow.provider;
    flowStatus.textContent = 'pending';
    flowStatus.className = 'status-badge status-pending';
    activeFlowSection.style.display = 'block';

    // Scroll to active flow
    activeFlowSection.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

// Handle OAuth redirect flow
function handleOAuthFlow() {
    flowContent.innerHTML = `
        <div class="oauth-instructions">
            <p>A new window will open to authenticate with ${currentFlow.provider}.</p>
            <p>Please complete the authentication in the opened window.</p>
            <div class="loading-spinner"></div>
            <p class="polling-info">Waiting for authentication to complete...</p>
        </div>
    `;

    // Open auth URL in new window
    if (currentFlow.authUrl) {
        window.open(currentFlow.authUrl, '_blank');
    }

    // Start polling for status
    startPolling();
}

// Handle Device flow
function handleDeviceFlow() {
    flowContent.innerHTML = `
        <div class="device-flow-instructions">
            <h4>Step 1: Visit the verification URL</h4>
            <a href="${currentFlow.verificationUrl}" target="_blank" class="verification-link">
                ${currentFlow.verificationUrl}
            </a>

            <h4>Step 2: Enter this code</h4>
            <div class="user-code">
                <code>${formatUserCode(currentFlow.userCode)}</code>
                <button class="btn btn-copy" onclick="copyUserCode()">Copy</button>
            </div>

            <h4>Step 3: Complete authentication</h4>
            <p>After entering the code, complete the authentication in your browser.</p>

            <div class="loading-spinner"></div>
            <p class="polling-info">Polling for completion (every ${currentFlow.interval}s)...</p>
        </div>
    `;

    // Start polling for status
    startPolling();
}

// Format user code for better readability (add spaces)
function formatUserCode(code) {
    if (!code) return '';
    return code.match(/.{1,4}/g).join(' ');
}

// Copy user code to clipboard
function copyUserCode() {
    if (currentFlow && currentFlow.userCode) {
        navigator.clipboard.writeText(currentFlow.userCode).then(() => {
            showMessage('Code copied to clipboard!', 'success');
        }).catch(() => {
            showMessage('Failed to copy code', 'error');
        });
    }
}

// Start polling for OAuth status
function startPolling() {
    // Clear any existing interval
    if (pollInterval) {
        clearInterval(pollInterval);
    }

    // Start polling
    pollInterval = setInterval(checkOAuthStatus, currentFlow.interval * 1000);

    // Initial check
    checkOAuthStatus();
}

// Check OAuth status
async function checkOAuthStatus() {
    if (!currentFlow || !currentFlow.state) {
        return;
    }

    const apiUrl = getApiUrl();
    const url = `${apiUrl}/oauth/status/${currentFlow.state}`;

    try {
        const headers = {};
        
        const managementKey = getManagementKey();
        if (managementKey) {
            headers['X-Management-Key'] = managementKey;
        }

        const response = await fetch(url, {
            headers: headers
        });

        if (!response.ok) {
            throw new Error('Failed to check OAuth status');
        }

        const data = await response.json();

        // Update status badge
        updateFlowStatus(data.status);

        // Handle different statuses
        switch (data.status) {
            case 'completed':
                handleFlowComplete();
                break;
            case 'failed':
                handleFlowFailed(data.error || 'Authentication failed');
                break;
            case 'cancelled':
                handleFlowCancelled();
                break;
            case 'pending':
                // Continue polling
                break;
            default:
                console.warn('Unknown status:', data.status);
        }

    } catch (error) {
        console.error('Error checking OAuth status:', error);
        // Don't show error to user on every poll failure
        // Only stop polling if it's a 404 (state not found)
        if (error.message.includes('404')) {
            handleFlowFailed('OAuth session not found');
        }
    }
}

// Update flow status badge
function updateFlowStatus(status) {
    flowStatus.textContent = status;
    flowStatus.className = 'status-badge';

    switch (status) {
        case 'completed':
            flowStatus.classList.add('status-completed');
            break;
        case 'failed':
            flowStatus.classList.add('status-failed');
            break;
        case 'cancelled':
            flowStatus.classList.add('status-cancelled');
            break;
        default:
            flowStatus.classList.add('status-pending');
    }
}

// Handle flow completion
function handleFlowComplete() {
    stopPolling();

    flowContent.innerHTML = `
        <div class="flow-complete">
            <div class="success-icon">✓</div>
            <h4>Authentication Successful!</h4>
            <p>Your ${currentFlow.provider} account has been successfully connected.</p>
        </div>
    `;

    showMessage(`Successfully connected to ${currentFlow.provider}!`, 'success');

    // Reset after 3 seconds
    setTimeout(() => {
        hideActiveFlow();
        currentFlow = null;
    }, 3000);
}

// Handle flow failure
function handleFlowFailed(errorMessage) {
    stopPolling();

    flowContent.innerHTML = `
        <div class="flow-error">
            <div class="error-icon">✗</div>
            <h4>Authentication Failed</h4>
            <p>${errorMessage}</p>
        </div>
    `;

    showMessage(`Authentication failed: ${errorMessage}`, 'error');

    // Allow user to dismiss
    cancelFlowBtn.textContent = 'Close';
    cancelFlowBtn.onclick = () => {
        hideActiveFlow();
        currentFlow = null;
        cancelFlowBtn.textContent = 'Cancel';
        cancelFlowBtn.onclick = cancelOAuthFlow;
    };
}

// Handle flow cancellation
function handleFlowCancelled() {
    stopPolling();

    flowContent.innerHTML = `
        <div class="flow-cancelled">
            <div class="cancelled-icon">○</div>
            <h4>Authentication Cancelled</h4>
            <p>The authentication flow was cancelled.</p>
        </div>
    `;

    showMessage('Authentication was cancelled', 'info');

    // Reset after 2 seconds
    setTimeout(() => {
        hideActiveFlow();
        currentFlow = null;
    }, 2000);
}

// Cancel OAuth flow
async function cancelOAuthFlow() {
    if (!currentFlow || !currentFlow.state) {
        hideActiveFlow();
        currentFlow = null;
        return;
    }

    const apiUrl = getApiUrl();
    const url = `${apiUrl}/oauth/cancel/${currentFlow.state}`;

    try {
        const headers = {};
        
        const managementKey = getManagementKey();
        if (managementKey) {
            headers['X-Management-Key'] = managementKey;
        }

        const response = await fetch(url, {
            method: 'POST',
            headers: headers
        });

        if (!response.ok) {
            throw new Error('Failed to cancel OAuth flow');
        }

        stopPolling();
        handleFlowCancelled();

    } catch (error) {
        console.error('Error cancelling OAuth flow:', error);
        // Even if cancel fails, hide the UI
        stopPolling();
        hideActiveFlow();
        currentFlow = null;
    }
}

// Stop polling
function stopPolling() {
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
    }
}

// Hide active flow section
function hideActiveFlow() {
    activeFlowSection.style.display = 'none';
    flowContent.innerHTML = '';
}

// Initialize on DOM load
document.addEventListener('DOMContentLoaded', init);

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    stopPolling();
});
