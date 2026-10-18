<script setup lang="ts">
import { ref } from 'vue';
import { Bell, ChevronRight, LogOut, Palette, Check, User } from 'lucide-vue-next';

const props = defineProps<{
  userName: string;
  userRole: string;
  theme: string;
}>();

const emit = defineEmits<{
  (e: 'switch-theme', theme: string): void;
  (e: 'logout'): void;
}>();

// 主题菜单
const themeOpen = ref(false);
const themeOptions = [
  { id: 'dark', label: '深色经典' },
  { id: 'editorial', label: '编辑杂志' },
  { id: 'hand-drawn', label: '手绘涂鸦' },
  { id: 'macos', label: 'macOS 毛玻璃' },
  { id: 'neo', label: '新野兽派' },
];

const selectTheme = (id: string) => {
  emit('switch-theme', id);
  themeOpen.value = false;
};

const handleLogout = () => {
  emit('logout');
};
</script>

<template>
  <header class="h-16 border-b border-[var(--border)] flex items-center justify-between px-8 bg-[var(--bg-app)]">
    <div class="flex items-center gap-4 text-sm text-[var(--text-muted)]">
      <span class="hover:text-[var(--text-primary)] transition-colors cursor-pointer">工作区</span>
      <ChevronRight class="w-4 h-4" />
      <span class="text-[var(--text-primary)]">订单智能识别</span>
    </div>
    <div class="flex items-center gap-6">
      <!-- 主题切换 -->
      <div class="relative">
        <button
          @click="themeOpen = !themeOpen"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-[var(--border)] hover:bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-all text-sm"
          title="切换主题"
        >
          <Palette class="w-4 h-4" />
          <span>{{ themeOptions.find(t => t.id === theme)?.label || '主题' }}</span>
        </button>
        <div v-if="themeOpen" class="absolute right-0 top-full mt-2 w-44 bg-[var(--bg-card)] border border-[var(--border-strong)] rounded-xl shadow-xl p-1.5 z-50">
          <button
            v-for="opt in themeOptions"
            :key="opt.id"
            @click="selectTheme(opt.id)"
            class="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm hover:bg-[var(--bg-hover)] transition-colors"
            :class="theme === opt.id ? 'text-[var(--accent)] font-medium' : 'text-[var(--text-secondary)]'"
          >
            <span>{{ opt.label }}</span>
            <Check v-if="theme === opt.id" class="w-4 h-4" />
          </button>
        </div>
      </div>

      <div class="w-px h-6 bg-[var(--border-strong)]"></div>

      <!-- 用户信息 + 退出 -->
      <div class="flex items-center gap-3 cursor-pointer group">
        <div class="text-right">
          <p class="text-sm font-medium text-[var(--text-primary)]">{{ userName }}</p>
          <p class="text-xs text-[var(--text-faint)]">{{ userRole }}</p>
        </div>
        <div class="w-10 h-10 rounded-full bg-gradient-to-tr from-[var(--accent)] to-[var(--teal)] flex items-center justify-center">
          <User class="w-5 h-5 text-[var(--text-on-accent)]" />
        </div>
        <button
          @click="handleLogout"
          class="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-[var(--text-muted)] hover:text-red-400 hover:bg-[var(--bg-hover)] transition-all text-sm"
          title="退出登录"
        >
          <LogOut class="w-4 h-4" />
          <span>退出</span>
        </button>
      </div>
    </div>
  </header>
</template>
