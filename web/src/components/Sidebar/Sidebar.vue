<script setup lang="ts">
import { Box, FileText, Layers, Calculator, Settings, LogOut } from 'lucide-vue-next';

defineProps<{
  activeTab: string;
  isSystemReady: boolean;
  statusMessage: string;
}>();

defineEmits<{
  (e: 'update:activeTab', tab: string): void;
}>();
</script>

<template>
  <aside class="w-64 bg-[var(--bg-sidebar)] border-r border-[var(--border)] flex flex-col justify-between py-6 h-full shrink-0">
    <div class="px-6">
      <div class="flex items-center gap-3 mb-10 px-2">
        <div class="w-8 h-8 bg-[var(--accent)] rounded-lg flex items-center justify-center">
          <Box class="w-5 h-5 text-[var(--text-on-accent)]" />
        </div>
        <h1 class="text-xl font-bold tracking-tight text-[var(--text-primary)]">OrderMaster</h1>
      </div>
      <nav class="space-y-1">
        <button 
          @click="$emit('update:activeTab', 'input')"
          class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group text-left"
          :class="activeTab === 'input' ? 'bg-[var(--accent-soft-bg)] text-[var(--accent)] border-r-[3px] border-[var(--accent)]' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'"
        >
          <FileText class="w-5 h-5" />
          <span class="font-medium">订单输入</span>
        </button>
        <button 
          @click="$emit('update:activeTab', 'details')"
          class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 text-left"
          :class="activeTab === 'details' ? 'bg-[var(--accent-soft-bg)] text-[var(--accent)] border-r-[3px] border-[var(--accent)]' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'"
        >
          <Layers class="w-5 h-5" />
          <span class="font-medium">订单详情</span>
        </button>
        <button 
          @click="$emit('update:activeTab', 'checkout')"
          class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 text-left"
          :class="activeTab === 'checkout' ? 'bg-[var(--accent-soft-bg)] text-[var(--accent)] border-r-[3px] border-[var(--accent)]' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'"
        >
          <Calculator class="w-5 h-5" />
          <span class="font-medium">账单结算</span>
        </button>
<!--        <button -->
<!--          @click="$emit('update:activeTab', 'settings')"-->
<!--          class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 text-left"-->
<!--          :class="activeTab === 'settings' ? 'bg-[var(--accent-soft-bg)] text-[var(--accent)] border-r-[3px] border-[var(--accent)]' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'"
<!--        >-->
<!--          <Settings class="w-5 h-5" />
<!--          <span class="font-medium">应用设置</span>
<!--        </button>-->
      </nav>
    </div>
    <div class="px-6">
      <div class="p-4 rounded-2xl bg-gradient-to-br from-[var(--accent-soft-bg)] to-[var(--teal-soft)] border border-[var(--accent)]/20 mb-6">
        <p class="text-xs text-[var(--accent-soft)] font-bold mb-1">系统状态</p>
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full animate-pulse" :class="isSystemReady ? 'bg-green-500' : 'bg-amber-500'"></span>
          <span class="text-sm text-[var(--text-secondary)]">{{ statusMessage }}</span>
        </div>
      </div>
      <!-- <div class="flex items-center gap-3 px-4 py-2 opacity-60 hover:opacity-100 cursor-pointer transition-opacity">
        <LogOut class="w-4 h-4" />
        <span class="text-sm">退出登录</span>
      </div> -->
    </div>
  </aside>
</template>
