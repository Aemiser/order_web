<script setup lang="ts">
import { ref } from 'vue';
import { Box, Loader2, AlertCircle } from 'lucide-vue-next';
import * as authApi from '../../api/authApi';
import { setSession } from '../../api/auth';

const emit = defineEmits<{
  (e: 'success'): void;
}>();

const username = ref('');
const password = ref('');
const loading = ref(false);
const errorMsg = ref('');

// 支持回车提交
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !loading.value) {
    handleLogin();
  }
};

const handleLogin = async () => {
  if (!username.value.trim() || !password.value) {
    errorMsg.value = '请输入用户名和密码';
    return;
  }
  loading.value = true;
  errorMsg.value = '';
  try {
    const result = await authApi.login(username.value.trim(), password.value);
    setSession(result.token, result.user);
    emit('success');
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : '登录失败，请稍后重试';
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="min-h-screen w-full flex items-center justify-center bg-[var(--bg-app)] text-[var(--text-primary)] font-sans px-4">
    <div class="w-full max-w-md">
      <div class="flex flex-col items-center mb-8">
        <div class="w-14 h-14 rounded-2xl bg-[var(--accent)] flex items-center justify-center shadow-lg shadow-[var(--accent-glow)] mb-4">
          <Box class="w-7 h-7 text-[var(--text-on-accent)]" />
        </div>
        <h1 class="text-2xl font-bold tracking-tight">OrderMaster</h1>
        <p class="text-sm text-[var(--text-secondary)] mt-1">订单识别与结账系统</p>
      </div>

      <div class="bg-[var(--bg-card)] border border-[var(--border)] rounded-2xl p-8 shadow-xl">
        <div v-if="errorMsg" class="mb-5 flex items-center gap-2 px-4 py-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          <AlertCircle class="w-4 h-4 shrink-0" />
          <span>{{ errorMsg }}</span>
        </div>

        <form class="space-y-5" @submit.prevent="handleLogin">
          <div class="space-y-2">
            <label class="text-xs font-semibold text-[var(--text-secondary)] tracking-wide">用户名</label>
            <input
              v-model="username"
              type="text"
              autocomplete="username"
              placeholder="请输入用户名"
              class="w-full px-4 py-3 rounded-xl bg-[var(--bg-input)] border border-[var(--border)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)] transition-all text-sm"
              @keydown="handleKeydown"
            />
          </div>
          <div class="space-y-2">
            <label class="text-xs font-semibold text-[var(--text-secondary)] tracking-wide">密码</label>
            <input
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="请输入密码"
              class="w-full px-4 py-3 rounded-xl bg-[var(--bg-input)] border border-[var(--border)] text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)] transition-all text-sm"
              @keydown="handleKeydown"
            />
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full h-11 rounded-xl bg-[var(--accent)] hover:bg-[var(--accent-hover)] disabled:opacity-60 disabled:cursor-not-allowed text-[var(--text-on-accent)] font-bold flex items-center justify-center gap-2 shadow-lg shadow-[var(--accent-glow)] transition-all text-sm"
          >
            <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
            <span>{{ loading ? '登录中...' : '登 录' }}</span>
          </button>
        </form>

        <p class="mt-6 text-center text-xs text-[var(--text-muted)]">账号由系统管理员创建，暂不支持自助注册</p>
      </div>
    </div>
  </div>
</template>
