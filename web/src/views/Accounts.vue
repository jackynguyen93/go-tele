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
        <h2>{{ showEditModal ? 'Edit Account' : 'Add Binance Account' }}</h2>

        <form @submit.prevent="saveAccount">
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
            <label class="checkbox-label">
              <input v-model="formData.is_testnet" type="checkbox">
              Use Testnet
            </label>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input v-model="formData.is_active" type="checkbox">
              Active
            </label>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input v-model="formData.is_default" type="checkbox">
              Set as Default
            </label>
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
      formData: {
        name: '',
        api_key: '',
        api_secret: '',
        is_testnet: true,
        is_active: true,
        is_default: false
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
        is_default: account.is_default
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
        if (this.showEditModal) {
          await axios.put(`/api/accounts/${this.formData.id}`, this.formData)
        } else {
          await axios.post('/api/accounts', this.formData)
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
      this.formData = {
        name: '',
        api_key: '',
        api_secret: '',
        is_testnet: true,
        is_active: true,
        is_default: false
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
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #16181c;
  padding: 30px;
  border-radius: 15px;
  width: 500px;
  max-width: 90%;
  border: 1px solid #2f3336;
}

.modal h2 {
  margin: 0 0 25px 0;
  color: #e7e9ea;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  color: #71767b;
  font-size: 14px;
  margin-bottom: 8px;
  font-weight: 500;
}

.form-group input[type="text"],
.form-group input[type="password"] {
  width: 100%;
  padding: 12px 15px;
  background: #0f1419;
  border: 1px solid #2f3336;
  border-radius: 8px;
  color: #e7e9ea;
  font-size: 16px;
}

.form-group input:focus {
  outline: none;
  border-color: #1d9bf0;
}

.checkbox-label {
  display: flex !important;
  align-items: center;
  cursor: pointer;
  color: #e7e9ea !important;
  font-size: 16px !important;
}

.checkbox-label input[type="checkbox"] {
  margin-right: 10px;
  width: 20px;
  height: 20px;
  cursor: pointer;
}

.form-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  margin-top: 30px;
}

.btn-secondary {
  padding: 10px 20px;
  background: #2f3336;
  color: #e7e9ea;
  border: none;
  border-radius: 20px;
  cursor: pointer;
  font-size: 16px;
}

.btn-secondary:hover {
  background: #3f4347;
}
</style>
