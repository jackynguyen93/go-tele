<template>
  <div class="dashboard">
    <h1>Dashboard</h1>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">Total Trades</div>
        <div class="stat-value">{{ stats.total_trades || 0 }}</div>
      </div>

      <div class="stat-card">
        <div class="stat-label">Win Rate</div>
        <div class="stat-value">{{ (stats.win_rate || 0).toFixed(1) }}%</div>
      </div>

      <div class="stat-card positive">
        <div class="stat-label">Total PnL</div>
        <div class="stat-value">${{ (stats.total_pnl || 0).toFixed(2) }}</div>
      </div>

      <div class="stat-card">
        <div class="stat-label">Open Positions</div>
        <div class="stat-value">{{ stats.open_positions || 0 }}</div>
      </div>
    </div>

    <div class="positions-section">
      <h2>Recent Positions</h2>
      <div class="positions-list">
        <div v-if="positions.length === 0" class="empty-state">
          No positions yet
        </div>
        <div v-for="pos in positions.slice(0, 10)" :key="pos.id" class="position-card">
          <div class="position-header">
            <span class="symbol">{{ pos.symbol }}</span>
            <span :class="['status', pos.status]">{{ pos.status }}</span>
          </div>
          <div class="position-details">
            <div class="detail">
              <span class="label">Entry:</span>
              <span class="value">${{ pos.entry_price.toFixed(4) }}</span>
            </div>
            <div class="detail">
              <span class="label">Leverage:</span>
              <span class="value">{{ pos.leverage }}x</span>
            </div>
            <div class="detail" v-if="pos.pnl">
              <span class="label">PnL:</span>
              <span :class="['value', pos.pnl > 0 ? 'positive' : 'negative']">
                ${{ pos.pnl.toFixed(2) }} ({{ pos.pnl_percent.toFixed(2) }}%)
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'Dashboard',
  data() {
    return {
      stats: {},
      positions: []
    }
  },
  mounted() {
    this.loadData()
    window.addEventListener('ws-message', this.handleWebSocketMessage)
  },
  beforeUnmount() {
    window.removeEventListener('ws-message', this.handleWebSocketMessage)
  },
  methods: {
    async loadData() {
      try {
        const [statsRes, positionsRes] = await Promise.all([
          axios.get('/api/stats'),
          axios.get('/api/positions?limit=10')
        ])
        this.stats = statsRes.data
        this.positions = positionsRes.data || []
      } catch (error) {
        console.error('Failed to load data:', error)
      }
    },
    handleWebSocketMessage(event) {
      const data = event.detail
      if (data.type === 'position_update' || data.type === 'initial') {
        this.loadData()
      }
    }
  }
}
</script>

<style scoped>
.dashboard h1 {
  font-size: 32px;
  margin-bottom: 30px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 40px;
}

.stat-card {
  background: #16181c;
  padding: 25px;
  border-radius: 15px;
  border: 1px solid #2f3336;
}

.stat-card.positive {
  border-color: #00ba7c;
}

.stat-label {
  color: #71767b;
  font-size: 14px;
  margin-bottom: 10px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #e7e9ea;
}

.positions-section h2 {
  font-size: 24px;
  margin-bottom: 20px;
}

.positions-list {
  display: grid;
  gap: 15px;
}

.position-card {
  background: #16181c;
  padding: 20px;
  border-radius: 15px;
  border: 1px solid #2f3336;
}

.position-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.symbol {
  font-size: 18px;
  font-weight: 600;
  color: #1d9bf0;
}

.status {
  padding: 5px 15px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.status.open {
  background: rgba(29, 155, 240, 0.2);
  color: #1d9bf0;
}

.status.closed {
  background: rgba(113, 118, 123, 0.2);
  color: #71767b;
}

.position-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 10px;
}

.detail {
  display: flex;
  justify-content: space-between;
}

.detail .label {
  color: #71767b;
  font-size: 14px;
}

.detail .value {
  color: #e7e9ea;
  font-weight: 500;
}

.detail .value.positive {
  color: #00ba7c;
}

.detail .value.negative {
  color: #f4212e;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #71767b;
  font-size: 16px;
}
</style>
