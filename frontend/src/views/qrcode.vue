<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { Events } from '@wailsio/runtime';
import QRCode from 'qrcode';

const qrURL = ref('');
const qrImageData = ref('');
const status = ref('等待二维码...');
const statusType = ref<'loading' | 'success' | 'error' | 'warning'>('loading');

let unsubscribeUpdate: (() => void) | null = null;
let unsubscribeStatus: (() => void) | null = null;

const generateQRCode = async (url: string) => {
  if (!url) {
    qrImageData.value = '';
    return;
  }
  try {
    const dataUrl = await QRCode.toDataURL(url, {
      width: 240,
      margin: 2,
      color: {
        dark: '#000000',
        light: '#ffffff'
      }
    });
    qrImageData.value = dataUrl;
  } catch (err) {
    console.error('Failed to generate QR code:', err);
    qrImageData.value = '';
  }
};

onMounted(async () => {
  unsubscribeUpdate = Events.On('qrcode-update', (data: { data: string }) => {
    qrURL.value = data.data;
    generateQRCode(data.data);
    status.value = '请使用微信扫描二维码';
    statusType.value = 'loading';
  });

  unsubscribeStatus = Events.On('qrcode-status', (data: { data: string }) => {
    const statusText = data.data;
    switch (statusText) {
      case 'scanned':
        status.value = '扫码成功，请在手机上确认登录';
        statusType.value = 'success';
        break;
      case 'confirmed':
        status.value = '登录成功';
        statusType.value = 'success';
        break;
      case 'expired':
        status.value = '二维码已过期，请重新启动';
        statusType.value = 'error';
        break;
      case 'error':
        status.value = '登录失败，请重试';
        statusType.value = 'error';
        break;
      default:
        status.value = statusText;
    }
  });
});

onUnmounted(() => {
  if (unsubscribeUpdate) {
    unsubscribeUpdate();
  }
  if (unsubscribeStatus) {
    unsubscribeStatus();
  }
});
</script>

<template>
  <div class="qrcode-container">
    <div class="qrcode-header">
      <h2>微信登录</h2>
    </div>
    
    <div class="qrcode-content">
      <div class="qrcode-image" v-if="qrImageData">
        <img :src="qrImageData" alt="微信登录二维码" />
      </div>
      <div class="qrcode-placeholder" v-else>
        <div class="loading-spinner"></div>
        <p>正在生成二维码...</p>
      </div>
    </div>

    <div class="qrcode-status" :class="statusType">
      <span class="status-icon" v-if="statusType === 'loading'">⏳</span>
      <span class="status-icon" v-else-if="statusType === 'success'">✅</span>
      <span class="status-icon" v-else-if="statusType === 'error'">❌</span>
      <span class="status-icon" v-else-if="statusType === 'warning'">⚠️</span>
      <span class="status-text">{{ status }}</span>
    </div>

    <div class="qrcode-footer" v-if="qrURL">
      <p class="url-hint">或复制链接在浏览器中打开：</p>
      <div class="url-display">{{ qrURL }}</div>
    </div>
  </div>
</template>

<style scoped>
.qrcode-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.qrcode-header {
  margin-bottom: 20px;
}

.qrcode-header h2 {
  color: white;
  font-size: 24px;
  font-weight: 600;
  margin: 0;
}

.qrcode-content {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.qrcode-image img {
  width: 240px;
  height: 240px;
  display: block;
}

.qrcode-placeholder {
  width: 240px;
  height: 240px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #666;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #f3f3f3;
  border-top: 3px solid #667eea;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.qrcode-status {
  margin-top: 20px;
  padding: 12px 24px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.95);
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.qrcode-status.loading {
  color: #666;
}

.qrcode-status.success {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
}

.qrcode-status.error {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

.qrcode-status.warning {
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.1);
}

.status-icon {
  font-size: 16px;
}

.status-text {
  font-weight: 500;
}

.qrcode-footer {
  margin-top: 16px;
  text-align: center;
}

.url-hint {
  color: rgba(255, 255, 255, 0.8);
  font-size: 12px;
  margin-bottom: 8px;
}

.url-display {
  background: rgba(255, 255, 255, 0.2);
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 11px;
  color: white;
  max-width: 280px;
  word-break: break-all;
}
</style>
