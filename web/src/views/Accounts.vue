<template>
  <div class="accounts">
    <div class="header">
      <h1>Binance Accounts</h1>
      <button class="btn-primary" @click="showAddModal = true">+ Add Account</button>
    </div>

    <div class="accounts-grid">
      <div v-for="account in accounts" :key="account.id" class="account-card">
        <div class="account-header">
          <div>
            <h3>{{ account.name }}</h3>
            <span class="badge" :class="{ testnet: account.is_testnet }">
              {{ account.is_testnet ? 'Testnet' : 'Production' }}
            </span>
            <span v-if="account.is_default" class="badge default">Default</span>
          </div>
          <div class="actions">
            <button class="btn-sm" @click="editAccount(account)">Edit</button>
            <button class="btn-sm btn-danger" @click="deleteAccount(account)">Delete</button>
          </div>
        </div>

        <div class="account-details">
          <div class="detail">
            <span class="label">API Key:</span>
            <span class="value mono">{{ account.api_key }}</span>
          </div>
          <div class="detail">
            <span class="label">API Secret:</span>
            <span class="value mono">{{ account.api_secret }}</span>
          </div>
          <div class="detail">
            <span class="label">Status:</span>
            <span :class="['status', account.is_active ? 'active' : 'inactive']">
              {{ account.is_active ? 'Active' : 'Inactive' }}
            </span>
          </div>
        </div>

        <div class="trading-config">
          <h4>Trading Configuration</h4>
          <div class="config-grid">
            <div class="config-item">
              <span class="config-label">Leverage:</span>
              <span class="config-value">{{ account.leverage }}x</span>
            </div>
            <div class="config-item">
              <span class="config-label">Order Amount:</span>
              <span class="config-value">${{ account.order_amount }}</span>
            </div>
            <div class="config-item">
              <span class="config-label">Target Profit:</span>
              <span class="config-value">{{ (account.target_percent * 100).toFixed(2) }}%</span>
            </div>
            <div class="config-item">
              <span class="config-label">Stop Loss:</span>
              <span class="config-value">{{ (account.stoploss_percent * 100).toFixed(2) }}%</span>
            </div>
            <div class="config-item">
              <span class="config-label">Order Timeout:</span>
              <span class="config-value">{{ account.order_timeout }}s</span>
            </div>
          </div>
        </div>

        <button v-if="!account.is_default" class="btn-set-default" @click="setDefault(account)">
          Set as Default
        </button>
      </div>

      <div v-if="accounts.length === 0" class="empty-state">
        <p>No Binance accounts configured</p>
        <p class="hint">Add your first account to start trading</p>
      </div>
    </div>

    <!-- Add/Edit Modal -->
    <div v-if="showAddModal || showEditModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <div class="modal-header">
          <h2>{{ showEditModal ? 'Edit Account' : 'Add Binance Account' }}</h2>
          <button class="modal-close" @click="closeModal">&times;</button>
        </div>

        <div class="modal-tabs">
          <button 
            :class="['tab', { active: activeTab === 'account' }]"
            @click="activeTab = 'account'"
          >
            Account Info
          </button>
          <button 
            :class="['tab', { active: activeTab === 'trading' }]"
            @click="activeTab = 'trading'"
          >
            Trading Config
          </button>
        </div>

        <form @submit.prevent="saveAccount" class="modal-form">
          <div v-show="activeTab === 'account'" class="tab-content">
            <div class="form-group">
              <label>Account Name *</label>
              <input
                v-model="formData.name"
                type="text"
                placeholder="e.g., Main Account, Testnet"
                required
              >
            </div>

            <div class="form-group">
              <label>API Key *</label>
              <input
                v-model="formData.api_key"
                type="text"
                placeholder="Your Binance API Key"
                required
              >
            </div>

            <div class="form-group">
              <label>API Secret *</label>
              <input
                v-model="formData.api_secret"
                type="password"
                placeholder="Your Binance API Secret"
                required
              >
            </div>

            <div class="form-group">
              <label class="section-label">Settings</label>
              <div class="checkbox-group">
                <label class="checkbox-label">
                  <input v-model="formData.is_testnet" type="checkbox">
                  <span>Use Testnet</span>
                </label>
                <label class="checkbox-label">
                  <input v-model="formData.is_active" type="checkbox">
                  <span>Active</span>
                </label>
                <label class="checkbox-label">
                  <input v-model="formData.is_default" type="checkbox">
                  <span>Set as Default</span>
                </label>
              </div>
            </div>
          </div>

          <div v-show="activeTab === 'trading'" class="tab-content">
            <div class="form-row">
              <div class="form-group">
                <label>Leverage</label>
                <input
                  v-model.number="formData.leverage"
                  type="number"
                  min="1"
                  max="125"
                  required
                >
              </div>

              <div class="form-group">
                <label>Order Amount (USDT)</label>
                <input
                  v-model.number="formData.order_amount"
                  type="number"
                  min="10"
                  step="10"
                  required
                >
              </div>
            </div>

            <div class="form-row">
              <div class="form-group">
                <label>Target Profit %</label>
                <input
                  v-model.number="formData.target_percent"
                  type="number"
                  step="0.01"
                  min="0.01"
                  required
                >
              </div>

              <div class="form-group">
                <label>Stop Loss %</label>
                <input
                  v-model.number="formData.stoploss_percent"
                  type="number"
                  step="0.01"
                  min="0.01"
                  required
                >
              </div>
            </div>

            <div class="form-group">
              <label>Order Timeout (seconds)</label>
              <input
                v-model.number="formData.order_timeout"
                type="number"
                min="60"
                required
              >
            </div>
          </div>

          <div class="form-actions">
            <button type="button" class="btn-secondary" @click="closeModal">Cancel</button>
            <button type="submit" class="btn-primary">
              {{ showEditModal ? 'Update' : 'Add' }} Account
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'Accounts',
  data() {
    return {
      accounts: [],
      showAddModal: false,
      showEditModal: false,
      activeTab: 'account',
      formData: {
        name: '',
        api_key: '',
        api_secret: '',
        is_testnet: true,
        is_active: true,
        is_default: false,
        leverage: 10,
        order_amount: 100,
        target_percent: 2,
        stoploss_percent: 1,
        order_timeout: 600
      }
    }
  },
  mounted() {
    this.loadAccounts()
  },
  methods: {
    async loadAccounts() {
      try {
        const res = await axios.get('/api/accounts')
        this.accounts = res.data || []
      } catch (error) {
        console.error('Failed to load accounts:', error)
      }
    },
    editAccount(account) {
      this.formData = {
        id: account.id,
        name: account.name,
        api_key: account.api_key,
        api_secret: '', // Don't pre-fill secret for security
        is_testnet: account.is_testnet,
        is_active: account.is_active,
        is_default: account.is_default,
        leverage: account.leverage || 10,
        order_amount: account.order_amount || 100,
        // Convert from decimal (0.02) to percentage (2) for the form
        target_percent: account.target_percent ? (account.target_percent * 100) : 2,
        stoploss_percent: account.stoploss_percent ? (account.stoploss_percent * 100) : 1,
        order_timeout: account.order_timeout || 600
      }
      this.showEditModal = true
    },
    async deleteAccount(account) {
      if (!confirm(`Delete account "${account.name}"?`)) return

      try {
        await axios.delete(`/api/accounts/${account.id}`)
        this.loadAccounts()
      } catch (error) {
        alert(error.response?.data?.error || 'Failed to delete account')
      }
    },
    async setDefault(account) {
      try {
        await axios.post(`/api/accounts/${account.id}/set-default`)
        this.loadAccounts()
      } catch (error) {
        alert('Failed to set default account')
      }
    },
    async saveAccount() {
      try {
        // Convert percentages from whole numbers to decimals
        const payload = {
          ...this.formData,
          target_percent: this.formData.target_percent / 100,
          stoploss_percent: this.formData.stoploss_percent / 100
        }

        if (this.showEditModal) {
          await axios.put(`/api/accounts/${payload.id}`, payload)
        } else {
          await axios.post('/api/accounts', payload)
        }
        this.loadAccounts()
        this.closeModal()
      } catch (error) {
        alert(error.response?.data?.error || 'Failed to save account')
      }
    },
    closeModal() {
      this.showAddModal = false
      this.showEditModal = false
      this.activeTab = 'account'
      this.formData = {
        name: '',
        api_key: '',
        api_secret: '',
        is_testnet: true,
        is_active: true,
        is_default: false,
        leverage: 10,
        order_amount: 100,
        target_percent: 2,
        stoploss_percent: 1,
        order_timeout: 600
      }
    }
  }
}
</script>

<style scoped>
.accounts {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
}

.header h1 {
  font-size: 32px;
  margin: 0;
}

.btn-primary {
  padding: 12px 24px;
  background: #1d9bf0;
  color: #fff;
  border: none;
  border-radius: 25px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
}

.btn-primary:hover {
  background: #1a8cd8;
}

.accounts-grid {
  display: grid;
  gap: 20px;
}

.account-card {
  background: #16181c;
  padding: 25px;
  border-radius: 15px;
  border: 1px solid #2f3336;
}

.account-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 20px;
}

.account-header h3 {
  font-size: 20px;
  margin: 0 0 10px 0;
  color: #e7e9ea;
}

.badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  margin-right: 8px;
}

.badge.testnet {
  background: rgba(255, 187, 0, 0.2);
  color: #ffbb00;
}

.badge.default {
  background: rgba(29, 155, 240, 0.2);
  color: #1d9bf0;
}

.actions {
  display: flex;
  gap: 10px;
}

.btn-sm {
  padding: 6px 14px;
  background: #2f3336;
  color: #e7e9ea;
  border: none;
  border-radius: 15px;
  cursor: pointer;
  font-size: 14px;
}

.btn-sm:hover {
  background: #3f4347;
}

.btn-sm.btn-danger {
  background: rgba(244, 33, 46, 0.2);
  color: #f4212e;
}

.btn-sm.btn-danger:hover {
  background: rgba(244, 33, 46, 0.3);
}

.account-details {
  margin-bottom: 20px;
}

.detail {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid #2f3336;
}

.detail .label {
  color: #71767b;
  font-size: 14px;
}

.detail .value {
  color: #e7e9ea;
  font-weight: 500;
}

.detail .value.mono {
  font-family: 'Courier New', monospace;
  font-size: 13px;
}

.status.active {
  color: #00ba7c;
}

.status.inactive {
  color: #71767b;
}

.btn-set-default {
  width: 100%;
  padding: 10px;
  background: rgba(29, 155, 240, 0.1);
  color: #1d9bf0;
  border: 1px solid #1d9bf0;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}

.btn-set-default:hover {
  background: rgba(29, 155, 240, 0.2);
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-state p {
  color: #71767b;
  font-size: 18px;
  margin: 10px 0;
}

.empty-state .hint {
  font-size: 14px;
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.modal {
  background: #16181c;
  border-radius: 20px;
  width: 600px;
  max-width: 95%;
  max-height: 90vh;
  border: 1px solid #2f3336;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 25px 30px 20px;
  border-bottom: 1px solid #2f3336;
}

.modal-header h2 {
  margin: 0;
  color: #e7e9ea;
  font-size: 24px;
  font-weight: 600;
}

.modal-close {
  background: none;
  border: none;
  color: #71767b;
  font-size: 32px;
  line-height: 1;
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  transition: all 0.2s;
}

.modal-close:hover {
  background: #2f3336;
  color: #e7e9ea;
}

.modal-tabs {
  display: flex;
  padding: 0 30px;
  border-bottom: 1px solid #2f3336;
  gap: 0;
}

.tab {
  padding: 16px 24px;
  background: none;
  border: none;
  color: #71767b;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
  position: relative;
  top: 1px;
}

.tab:hover {
  color: #e7e9ea;
  background: rgba(29, 155, 240, 0.05);
}

.tab.active {
  color: #1d9bf0;
  border-bottom-color: #1d9bf0;
}

.modal-form {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.tab-content {
  padding: 25px 30px;
  overflow-y: auto;
  flex: 1;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  color: #e7e9ea;
  font-size: 14px;
  margin-bottom: 8px;
  font-weight: 500;
}

.form-group input[type="text"],
.form-group input[type="password"],
.form-group input[type="number"] {
  width: 100%;
  padding: 12px 15px;
  background: #0f1419;
  border: 1px solid #2f3336;
  border-radius: 10px;
  color: #e7e9ea;
  font-size: 15px;
  transition: all 0.2s;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #1d9bf0;
  background: #1a1f24;
}

.form-group input:hover {
  border-color: #3f4347;
}

.section-label {
  color: #71767b !important;
  font-size: 13px !important;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 12px !important;
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: #0f1419;
  padding: 16px;
  border-radius: 10px;
  border: 1px solid #2f3336;
}

.checkbox-label {
  display: flex !important;
  align-items: center;
  cursor: pointer;
  color: #e7e9ea !important;
  font-size: 15px !important;
  margin: 0 !important;
  transition: color 0.2s;
}

.checkbox-label:hover {
  color: #fff !important;
}

.checkbox-label input[type="checkbox"] {
  margin-right: 12px;
  width: 18px;
  height: 18px;
  cursor: pointer;
  accent-color: #1d9bf0;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.form-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  padding: 20px 30px;
  border-top: 1px solid #2f3336;
  background: #16181c;
  border-radius: 0 0 20px 20px;
}

.btn-secondary {
  padding: 12px 24px;
  background: #2f3336;
  color: #e7e9ea;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 500;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: #3f4347;
}
</style>
