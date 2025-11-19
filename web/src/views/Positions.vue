<template>
  <div class="positions">
    <h1>Positions</h1>

    <div class="filters">
      <button :class="{ active: filter === 'all' }" @click="filter = 'all'">All</button>
      <button :class="{ active: filter === 'open' }" @click="filter = 'open'">Open</button>
      <button :class="{ active: filter === 'closed' }" @click="filter = 'closed'">Closed</button>
    </div>

    <div class="positions-table">
      <table>
        <thead>
          <tr>
            <th>Symbol</th>
            <th>Side</th>
            <th>Entry Price</th>
            <th>Quantity</th>
            <th>Leverage</th>
            <th>TP Price</th>
            <th>SL Price</th>
            <th>Status</th>
            <th>PnL</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="pos in filteredPositions" :key="pos.id">
            <td class="symbol">{{ pos.symbol }}</td>
            <td>{{ pos.side }}</td>
            <td>${{ pos.entry_price.toFixed(4) }}</td>
            <td>{{ pos.quantity.toFixed(4) }}</td>
            <td>{{ pos.leverage }}x</td>
            <td>${{ pos.take_profit_price.toFixed(4) }}</td>
            <td>${{ pos.stop_loss_price.toFixed(4) }}</td>
            <td><span :class="['badge', pos.status]">{{ pos.status }}</span></td>
            <td>
              <span v-if="pos.pnl" :class="['pnl', pos.pnl > 0 ? 'positive' : 'negative']">
                ${{ pos.pnl.toFixed(2) }} ({{ pos.pnl_percent.toFixed(2) }}%)
              </span>
              <span v-else>-</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'Positions',
  data() {
    return {
      positions: [],
      filter: 'all'
    }
  },
  computed: {
    filteredPositions() {
      if (this.filter === 'all') return this.positions
      return this.positions.filter(p => p.status === this.filter)
    }
  },
  mounted() {
    this.loadPositions()
    window.addEventListener('ws-message', this.handleWebSocketMessage)
  },
  beforeUnmount() {
    window.removeEventListener('ws-message', this.handleWebSocketMessage)
  },
  methods: {
    async loadPositions() {
      try {
        const res = await axios.get('/api/positions?limit=100')
        this.positions = res.data || []
      } catch (error) {
        console.error('Failed to load positions:', error)
      }
    },
    handleWebSocketMessage(event) {
      const data = event.detail
      if (data.type === 'position_update') {
        this.loadPositions()
      }
    }
  }
}
</script>

<style scoped>
.positions h1 {
  font-size: 32px;
  margin-bottom: 30px;
}

.filters {
  display: flex;
  gap: 10px;
  margin-bottom: 30px;
}

.filters button {
  padding: 10px 20px;
  background: #16181c;
  color: #e7e9ea;
  border: 1px solid #2f3336;
  border-radius: 20px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.filters button:hover {
  background: #1d1f23;
}

.filters button.active {
  background: #1d9bf0;
  border-color: #1d9bf0;
  font-weight: 600;
}

.positions-table {
  background: #16181c;
  border-radius: 15px;
  padding: 20px;
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th {
  text-align: left;
  padding: 15px;
  color: #71767b;
  font-size: 14px;
  font-weight: 600;
  border-bottom: 1px solid #2f3336;
}

td {
  padding: 15px;
  color: #e7e9ea;
  border-bottom: 1px solid #2f3336;
}

tr:hover {
  background: rgba(29, 155, 240, 0.05);
}

.symbol {
  color: #1d9bf0;
  font-weight: 600;
}

.badge {
  padding: 5px 12px;
  border-radius: 15px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.badge.open {
  background: rgba(29, 155, 240, 0.2);
  color: #1d9bf0;
}

.badge.closed {
  background: rgba(113, 118, 123, 0.2);
  color: #71767b;
}

.pnl {
  font-weight: 600;
}

.pnl.positive {
  color: #00ba7c;
}

.pnl.negative {
  color: #f4212e;
}
</style>
