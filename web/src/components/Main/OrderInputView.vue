<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import OrderInput from './OrderInput.vue';
import OrderTable from './OrderTable.vue';
import type { Order } from '../../types';

const props = defineProps<{
  inputText: string;
  isAnalyzing: boolean;
  orders: Order[];
}>();

const emit = defineEmits<{
  (e: 'update:inputText', value: string): void;
  (e: 'clear'): void;
  (e: 'analyze', isAppend: boolean, modelId: number): void;
  (e: 'removeOrder', id: string): void;
  (e: 'clearAllOrders'): void;
  (e: 'refreshData'): void;
  (e: 'delete', id: string): void;
}>();

// 全屏状态检测
const isFullscreen = ref(false);
const isCollapsed = ref(false);

// 检测全屏状态
const checkFullscreen = () => {
  const fullscreenElement = document.fullscreenElement || 
                          (document as any).webkitFullscreenElement || 
                          (document as any).mozFullScreenElement || 
                          (document as any).msFullscreenElement;
  
  const wasFullscreen = isFullscreen.value;
  isFullscreen.value = !!fullscreenElement;
  
  // 当进入全屏时收缩区域，退出全屏时展开区域
  if (isFullscreen.value && !wasFullscreen) {
    isCollapsed.value = true;
  } else if (!isFullscreen.value && wasFullscreen) {
    isCollapsed.value = false;
  }
};

// 监听全屏变化事件
onMounted(() => {
  document.addEventListener('fullscreenchange', checkFullscreen);
  document.addEventListener('webkitfullscreenchange', checkFullscreen);
  document.addEventListener('mozfullscreenchange', checkFullscreen);
  document.addEventListener('MSFullscreenChange', checkFullscreen);
});

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', checkFullscreen);
  document.removeEventListener('webkitfullscreenchange', checkFullscreen);
  document.removeEventListener('mozfullscreenchange', checkFullscreen);
  document.removeEventListener('MSFullscreenChange', checkFullscreen);
});
</script>

<template>
  <div class="space-y-6">
    <div :class="['transition-all duration-500 ease-in-out', isCollapsed ? 'max-h-40 overflow-hidden' : 'max-h-none']">
      <OrderInput 
        :inputText="inputText"
        :isAnalyzing="isAnalyzing"
        @update:inputText="$emit('update:inputText', $event)"
        @clear="$emit('clear')"
        @analyze="(isAppend: boolean, modelId: number) => $emit('analyze', isAppend, modelId)"
      />
    </div>

    <div :class="['transition-all duration-500 ease-in-out', isCollapsed ? 'opacity-100' : 'opacity-100']">
      <OrderTable 
        :orders="orders"
        @remove="$emit('removeOrder', $event)"
        @clearAll="$emit('clearAllOrders')"
        @refreshData="$emit('refreshData')"
        @delete="$emit('delete', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
/* 全屏状态下的样式调整 */
:deep(.order-input) {
  transition: all 0.5s ease-in-out;
}

:deep(.order-table) {
  transition: all 0.5s ease-in-out;
}
</style>
