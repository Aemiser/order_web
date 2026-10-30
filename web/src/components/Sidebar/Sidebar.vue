<script setup lang="ts">
import { Box, FileText, Layers, Calculator, Milk, Citrus, Tags } from 'lucide-vue-next';

defineProps<{
  activeTab: string;
  isSystemReady: boolean;
  statusMessage: string;
  brand: 'ydd' | 'wy';
}>();

const emit = defineEmits<{
  (e: 'update:activeTab', tab: string): void;
  (e: 'update:brand', brand: 'ydd' | 'wy'): void;
}>();

// 双品牌：一点点（原有链路）/ 吴饮（吾饮良品，独立库+仓库标签）。
// 切换品牌由 App 重置页签到该品牌首页。
const switchBrand = (b: 'ydd' | 'wy') => emit('update:brand', b);

const navItemClass = (active: boolean) =>
  [
    'w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 text-left',
    active
      ? 'bg-[var(--accent-soft-bg)] text-[var(--accent)] border-r-[3px] border-[var(--accent)] shadow-sm shadow-[var(--accent-glow)]'
      : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]',
  ].join(' ');
</script>

<template>
  <aside class="w-64 bg-[var(--bg-sidebar)] border-r border-[var(--border)] flex flex-col justify-between py-6 h-full shrink-0">
    <div class="px-6">
      <div class="flex items-center gap-3 mb-6 px-2">
        <div class="w-8 h-8 bg-[var(--accent)] rounded-lg flex items-center justify-center">
          <Box class="w-5 h-5 text-[var(--text-on-accent)]" />
        </div>
        <h1 class="text-xl font-bold tracking-tight text-[var(--text-primary)]">OrderMaster</h1>
      </div>

      <!-- 品牌切换 -->
      <div class="grid grid-cols-2 gap-1 p-1 mb-6 rounded-xl bg-[var(--bg-hover)]">
        <button
          @click="switchBrand('ydd')"
          class="flex items-center justify-center gap-1.5 py-2 rounded-lg text-xs font-bold transition-all"
          :class="brand === 'ydd' ? 'bg-[var(--bg-card)] text-[var(--accent)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'"
        >
          <Milk class="w-3.5 h-3.5" /> 一点点
        </button>
        <button
          @click="switchBrand('wy')"
          class="flex items-center justify-center gap-1.5 py-2 rounded-lg text-xs font-bold transition-all"
          :class="brand === 'wy' ? 'bg-[var(--bg-card)] text-[var(--accent)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'"
        >
          <Citrus class="w-3.5 h-3.5" /> 吴饮
        </button>
      </div>

      <nav class="space-y-2 mt-2">
        <p class="px-4 mb-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--text-faint)]">工作台</p>

        <!-- 一点点导航 -->
        <template v-if="brand === 'ydd'">
          <button @click="$emit('update:activeTab', 'input')" :class="navItemClass(activeTab === 'input')">
            <FileText class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">订单输入</span>
          </button>
          <button @click="$emit('update:activeTab', 'details')" :class="navItemClass(activeTab === 'details')">
            <Layers class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">订单详情</span>
          </button>
          <button @click="$emit('update:activeTab', 'checkout')" :class="navItemClass(activeTab === 'checkout')">
            <Calculator class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">账单结算</span>
          </button>
        </template>

        <!-- 吴饮导航 -->
        <template v-else>
          <button @click="$emit('update:activeTab', 'wy-input')" :class="navItemClass(activeTab === 'wy-input')">
            <FileText class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">订单输入</span>
          </button>
          <button @click="$emit('update:activeTab', 'wy-details')" :class="navItemClass(activeTab === 'wy-details')">
            <Layers class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">订单详情</span>
          </button>
          <button @click="$emit('update:activeTab', 'wy-checkout')" :class="navItemClass(activeTab === 'wy-checkout')">
            <Calculator class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">标签结账</span>
          </button>
          <button @click="$emit('update:activeTab', 'wy-settings')" :class="navItemClass(activeTab === 'wy-settings')">
            <Tags class="w-5 h-5 shrink-0" />
            <span class="font-medium truncate">标签管理</span>
          </button>
        </template>
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
    </div>
  </aside>
</template>
