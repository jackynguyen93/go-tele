<template>
  <div class="settings">
    <h1>Settings</h1>

    <div class="settings-card">
      <h2>Trading Configuration</h2>

      <div class="form-group">
        <label>
          <input type="checkbox" v-model="config.enabled" @change="saveConfig">
          Trading Enabled
        </label>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label>Leverage</label>
          <input
            type="number"
            v-model.number="config.leverage"
            @blur="saveConfig"
            min="1"
            max="125"
          >
        </div>

        <div class="form-group">
          <label>Order Amount (USDT)</label>
          <input
            type="number"
            v-model.number="config.order_amount"
            @blur="saveConfig"
            min="10"
            step="10"
          >
        </div>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label>Target Profit %</label>
          <input
            type="number"
            v-model.number="config.target_percent"
            @blur="saveConfig"
            step="0.01"
            min="0.01"
          >
        </div>

        <div class="form-group">
          <label>Stop Loss %</label>
          <input
            type="number"
            v-model.number="config.stoploss_percent"
            @blur="saveConfig"
            step="0.01"
            min="0.01"
          >
        </div>
      </div>

      <div class="form-group">
        <label>Order Timeout (seconds)</label>
        <input
          type="number"
          v-model.number="config.order_timeout"
          @blur="saveConfig"
          min="60"
        >
      </div>

      <div class="form-group">
        <label>Signal Pattern (Regex)</label>
        <input
          type="text"
          v-model="config.signal_pattern"
          @blur="saveConfig"
          placeholder="e.g., \$([A-Z]{2,10})"
        >
      </div>

      <div class="form-group">
        <label>Ignore Tokens (Comma-separated)</label>
        <input
          type="text"
          v-model="config.ignore_tokens"
          @blur="saveConfig"
          placeholder="e.g., BTC, ETH, SOL (or BTCUSDT, ETHUSDT, SOLUSDT)"
        >
        <small style="color: #71767b; font-size: 12px; margin-top: 5px; display: block;">
          Enter tokens to ignore. You can use either "BTC" or "BTCUSDT" format. Tokens in this list will not trigger orders.
        </small>
      </div>

      <div v-if="saveMessage" class="save-message">
        {{ saveMessage }}
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'Settings',
  data() {
    return {
      config: {
        enabled: false,
        leverage: 10,
        order_amount: 100,
        target_percent: 0.02,
        stoploss_percent: 0.01,
        order_timeout: 600,
        signal_pattern: '',
        ignore_tokens: ''
      },
      saveMessage: ''
    }
  },
  mounted() {
    this.loadConfig()
  },
  methods: {
    async loadConfig() {
      try {
        const res = await axios.get('/api/config')
        if (res.data.trading) {
          this.config = { ...this.config, ...res.data.trading }
        }
      } catch (error) {
        console.error('Failed to load config:', error)
      }
    },
    async saveConfig() {
      try {
        await axios.put('/api/config', {
          trading: this.config
        })
        this.saveMessage = 'Settings saved successfully!'
        setTimeout(() => {
          this.saveMessage = ''
        }, 3000)
      } catch (error) {
        console.error('Failed to save config:', error)
        this.saveMessage = 'Failed to save settings'
      }
    }
  }
}
</script>

<style scoped>
.settings h1 {
  font-size: 32px;
  margin-bottom: 30px;
}

.settings-card {
  background: #16181c;
  padding: 30px;
  border-radius: 15px;
  border: 1px solid #2f3336;
  max-width: 800px;
}

.settings-card h2 {
  font-size: 24px;
  margin-bottom: 25px;
  color: #e7e9ea;
}

.form-group {
  margin-bottom: 20px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

label {
  display: block;
  color: #71767b;
  font-size: 14px;
  margin-bottom: 8px;
  font-weight: 500;
}

input[type="text"],
input[type="number"] {
  width: 100%;
  padding: 12px 15px;
  background: #0f1419;
  border: 1px solid #2f3336;
  border-radius: 8px;
  color: #e7e9ea;
  font-size: 16px;
}

input[type="text"]:focus,
input[type="number"]:focus {
  outline: none;
  border-color: #1d9bf0;
}

input[type="checkbox"] {
  margin-right: 10px;
  width: 20px;
  height: 20px;
  cursor: pointer;
}

label:has(input[type="checkbox"]) {
  display: flex;
  align-items: center;
  cursor: pointer;
  color: #e7e9ea;
  font-size: 16px;
}

.save-message {
  margin-top: 20px;
  padding: 15px;
  background: rgba(0, 186, 124, 0.2);
  color: #00ba7c;
  border-radius: 8px;
  text-align: center;
}
</style>
