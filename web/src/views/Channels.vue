<template>
  <div class="channels">
    <div class="header">
      <h1>Telegram Channels</h1>
      <button class="btn-primary" @click="showAddModal = true">+ Subscribe</button>
    </div>

    <div class="channels-grid">
      <div v-for="channel in channels" :key="channel.id" class="channel-card">
        <div class="channel-header">
          <div>
            <h3>{{ channel.title }}</h3>
            <span class="channel-username" v-if="channel.username">@{{ channel.username }}</span>
            <span v-if="channel.is_active" class="badge active">Active</span>
            <span v-else class="badge inactive">Inactive</span>
          </div>
          <div class="actions">
            <button class="btn-sm btn-danger" @click="unsubscribe(channel)">Unsubscribe</button>
          </div>
        </div>

        <div class="channel-details">
          <div class="detail">
            <span class="label">Channel ID:</span>
            <span class="value mono">{{ channel.channel_id }}</span>
          </div>
          <div class="detail">
            <span class="label">Username:</span>
            <span class="value">{{ channel.username || 'N/A' }}</span>
          </div>
          <div class="detail">
            <span class="label">Subscribed:</span>
            <span class="value">{{ formatDate(channel.created_at) }}</span>
          </div>
        </div>
      </div>

      <div v-if="channels.length === 0" class="empty-state">
        <p>No channels subscribed</p>
        <p class="hint">Add your first channel to start monitoring</p>
      </div>
    </div>

    <!-- Add Modal -->
    <div v-if="showAddModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <h2>Subscribe to Channel</h2>

        <form @submit.prevent="subscribe">
          <div class="form-group">
            <label>Channel Identifier *</label>
            <input
              v-model="formData.identifier"
              type="text"
              placeholder="@username, channel ID, or invite link"
              required
            >
            <p class="form-hint">
              Examples:<br>
              - Username: telegram or @telegram<br>
              - Channel ID: -1002233859472<br>
              - Invite link: https://t.me/+wIr66-O-XaxjOWI0
            </p>
          </div>

          <div class="form-actions">
            <button type="button" class="btn-secondary" @click="closeModal">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="loading">
              {{ loading ? 'Subscribing...' : 'Subscribe' }}
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
  name: 'Channels',
  data() {
    return {
      channels: [],
      showAddModal: false,
      loading: false,
      formData: {
        identifier: ''
      }
    }
  },
  mounted() {
    this.loadChannels()
  },
  methods: {
    async loadChannels() {
      try {
        const res = await axios.get('/api/channels')
        this.channels = res.data || []
      } catch (error) {
        console.error('Failed to load channels:', error)
      }
    },
    async subscribe() {
      this.loading = true
      try {
        // Remove @ prefix if present
        const identifier = this.formData.identifier.replace(/^@/, '')

        await axios.post('/api/channels', { identifier })
        await this.loadChannels()
        this.closeModal()
        alert('Successfully subscribed to channel!')
      } catch (error) {
        alert(error.response?.data?.error || 'Failed to subscribe to channel')
      } finally {
        this.loading = false
      }
    },
    async unsubscribe(channel) {
      if (!confirm(`Unsubscribe from "${channel.title}"?`)) return

      try {
        await axios.delete(`/api/channels/${channel.channel_id}`)
        await this.loadChannels()
        alert('Successfully unsubscribed from channel')
      } catch (error) {
        alert(error.response?.data?.error || 'Failed to unsubscribe from channel')
      }
    },
    closeModal() {
      this.showAddModal = false
      this.formData = {
        identifier: ''
      }
    },
    formatDate(dateStr) {
      if (!dateStr) return 'N/A'
      const date = new Date(dateStr)
      return date.toLocaleString()
    }
  }
}
</script>

<style scoped>
.channels {
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

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.channels-grid {
  display: grid;
  gap: 20px;
}

.channel-card {
  background: #16181c;
  padding: 25px;
  border-radius: 15px;
  border: 1px solid #2f3336;
}

.channel-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 20px;
}

.channel-header h3 {
  font-size: 20px;
  margin: 0 0 10px 0;
  color: #e7e9ea;
}

.channel-username {
  display: block;
  color: #71767b;
  font-size: 14px;
  margin-bottom: 8px;
}

.badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
  margin-right: 8px;
}

.badge.active {
  background: rgba(0, 186, 124, 0.2);
  color: #00ba7c;
}

.badge.inactive {
  background: rgba(113, 118, 123, 0.2);
  color: #71767b;
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

.channel-details {
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

.form-group input[type="text"] {
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

.form-hint {
  color: #71767b;
  font-size: 12px;
  margin-top: 8px;
  line-height: 1.5;
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
