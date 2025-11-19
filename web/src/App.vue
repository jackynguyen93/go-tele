<template>
  <div class="app">
    <nav class="sidebar">
      <div class="logo">
        <h2>Trading Bot</h2>
      </div>
      <ul class="nav-links">
        <li>
          <router-link to="/" class="nav-link">
            <span class="icon">📊</span>
            <span>Dashboard</span>
          </router-link>
        </li>
        <li>
          <router-link to="/positions" class="nav-link">
            <span class="icon">💼</span>
            <span>Positions</span>
          </router-link>
        </li>
        <li>
          <router-link to="/accounts" class="nav-link">
            <span class="icon">🔑</span>
            <span>Accounts</span>
          </router-link>
        </li>
        <li>
          <router-link to="/settings" class="nav-link">
            <span class="icon">⚙️</span>
            <span>Settings</span>
          </router-link>
        </li>
      </ul>
      <div class="status">
        <div class="status-indicator" :class="{ active: isConnected }"></div>
        <span>{{ isConnected ? 'Connected' : 'Disconnected' }}</span>
      </div>
    </nav>
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      isConnected: false,
      ws: null
    }
  },
  mounted() {
    this.connectWebSocket()
  },
  beforeUnmount() {
    if (this.ws) {
      this.ws.close()
    }
  },
  methods: {
    connectWebSocket() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//${window.location.host}/api/ws`

      this.ws = new WebSocket(wsUrl)

      this.ws.onopen = () => {
        this.isConnected = true
        console.log('WebSocket connected')
      }

      this.ws.onclose = () => {
        this.isConnected = false
        console.log('WebSocket disconnected')
        // Reconnect after 5 seconds
        setTimeout(() => this.connectWebSocket(), 5000)
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      this.ws.onmessage = (event) => {
        const data = JSON.parse(event.data)
        this.handleWebSocketMessage(data)
      }
    },
    handleWebSocketMessage(data) {
      // Emit custom event that components can listen to
      window.dispatchEvent(new CustomEvent('ws-message', { detail: data }))
    }
  }
}
</script>

<style>
.app {
  display: flex;
  height: 100vh;
}

.sidebar {
  width: 250px;
  background: #16181c;
  padding: 20px;
  display: flex;
  flex-direction: column;
}

.logo h2 {
  color: #1d9bf0;
  margin-bottom: 30px;
  font-size: 24px;
}

.nav-links {
  list-style: none;
  flex: 1;
}

.nav-link {
  display: flex;
  align-items: center;
  padding: 15px 20px;
  color: #e7e9ea;
  text-decoration: none;
  border-radius: 25px;
  margin-bottom: 10px;
  transition: background 0.2s;
}

.nav-link:hover {
  background: #1d1f23;
}

.nav-link.router-link-active {
  background: #1d9bf0;
  font-weight: 600;
}

.nav-link .icon {
  margin-right: 15px;
  font-size: 20px;
}

.status {
  display: flex;
  align-items: center;
  padding: 15px;
  background: #1d1f23;
  border-radius: 10px;
  font-size: 14px;
}

.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #71767b;
  margin-right: 10px;
}

.status-indicator.active {
  background: #00ba7c;
  box-shadow: 0 0 10px #00ba7c;
}

.content {
  flex: 1;
  padding: 30px;
  overflow-y: auto;
  background: #0f1419;
}
</style>
