<script setup lang="ts">
import { Bell, ChevronRight } from 'lucide-vue-next';

defineProps<{
  userName: string;
  userRole: string;
}>();
</script>

<template>
  <header class="h-16 border-b border-white/5 flex items-center justify-between px-8 bg-[#0F1115]">
    <div class="flex items-center gap-4 text-sm text-slate-400">
      <span class="hover:text-white transition-colors cursor-pointer">工作区</span>
      <ChevronRight class="w-4 h-4" />
      <span class="text-white">订单智能识别</span>
    </div>
    <div class="flex items-center gap-6">
      <div class="relative">
        <Bell class="w-5 h-5 text-slate-400 hover:text-white cursor-pointer" />
        <span class="absolute -top-1 -right-1 w-2 h-2 bg-red-500 rounded-full border-2 border-[#0F1115]"></span>
      </div>
      <div class="flex items-center gap-3 pl-6 border-l border-white/10 cursor-pointer">
        <div class="text-right">
          <p class="text-sm font-medium">{{ userName }}</p>
          <p class="text-xs text-slate-500">{{ userRole }}</p>
        </div>
        <div class="w-10 h-10 rounded-full bg-gradient-to-tr from-blue-500 to-teal-400"></div>
      </div>
    </div>
  </header>
</template>
