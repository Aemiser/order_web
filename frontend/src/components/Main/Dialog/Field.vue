<template>
  <div v-if="show" class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 transition-opacity duration-300">
    <!-- Modal Container -->
    <div class="bg-[#16191E] border border-white/10 rounded-2xl p-8 max-w-5xl w-full h-[80vh] mx-4 transform transition-all duration-300 scale-100 flex flex-col overflow-hidden">
      <!-- Modal Body -->
      <div class="flex-1 flex overflow-hidden p-4 gap-4">
        <!-- Left Panel: Drag Source & L1 Creator -->
        <aside class="w-56 flex flex-col gap-3">
          <!-- + 添加一级元素 Drop Zone (TOP) -->
          <div 
            id="l1-drop-zone" 
            class="h-[46px] rounded-xl border-2 border-dashed border-blue-500/20 bg-blue-500/5 flex items-center justify-center gap-2 group hover:border-blue-500/50 hover:bg-blue-500/10 transition-all cursor-pointer shadow-inner"
            @dragover.prevent="handleDragOver($event, 'l1')"
            @dragleave="handleDragLeave($event, 'l1')"
            @drop="handleDrop($event, 'l1')"
          >
            <Icon icon="lucide:plus" class="text-blue-500 text-base group-hover:scale-110 transition-transform"></Icon>
            <p class="text-[11px] font-bold text-blue-400">添加一级元素</p>
          </div>
          
          <!-- Draggable Field List -->
          <div class="flex flex-col flex-1 gap-2 overflow-hidden">
            <div class="flex items-center justify-between px-1">
              <div class="flex items-center gap-2">
                <h3 class="text-[11px] font-display font-bold text-slate-500 uppercase tracking-widest">未分类字段</h3>
              </div>
            </div>
            <div class="flex-1 overflow-y-auto custom-scrollbar space-y-2 pr-1">
              <div 
                v-for="(field, index) in localFieldList" 
                :key="index"
                draggable="true"
                class="drag-item p-2.5 rounded-xl glass-card border border-white/5 flex items-center justify-between group hover:border-blue-500/30 hover:bg-white/5 transition-all cursor-grab"
                @dragstart="handleDragStart($event, field)"
                @dragend="handleDragEnd"
              >
                <div class="flex items-center gap-2.5">
                  <Icon icon="lucide:file-text" class="text-slate-500 group-hover:text-blue-500 transition-colors text-sm"></Icon>
                  <span class="text-[13px] font-medium text-slate-300">{{ field }}</span>
                </div>
                <Icon icon="lucide:grip-vertical" class="text-slate-700 opacity-0 group-hover:opacity-100 transition-opacity text-xs"></Icon>
              </div>
            </div>
          </div>
        </aside>
        
        <!-- Right Panel: Configuration Area -->
        <section class="flex-1 bg-[#0F1115] rounded-2xl border border-white/5 overflow-hidden flex flex-col shadow-inner">
          <div class="flex-1 flex overflow-x-auto custom-scrollbar">
            <!-- Dynamic Categories -->
            <div 
              v-for="(subFields, category) in localPbMap" 
              :key="category"
              class="w-8 min-w-[12rem] border-r border-white/5 flex flex-col bg-white/[0.01]"
            >
              <div class="p-3.5 flex items-center justify-between bg-[#1E2228] border-b border-white/10">
                <div class="flex items-center gap-2">
                  <span class="w-1 h-4 bg-blue-600 rounded-full"></span>
                  <h4 class="font-display font-bold text-[13px] text-blue-100">{{ category }}</h4>
                </div>
                <button class="text-red-500 hover:text-red-400 transition-all duration-200 p-2 rounded-full bg-red-400/10 hover:bg-red-400/20" @click="removeCategory(category)">
                  <Icon icon="lucide:trash-2" class="text-sm transition-transform duration-200 hover:scale-110"></Icon>
                </button>
              </div>
              <div class="p-3 flex-1 space-y-2.5">
                <!-- Level-2 Drop Zone -->
                <div 
                  class="l2-drop-zone h-10 rounded-lg border border-dashed border-white/10 flex items-center justify-center gap-2 text-slate-600 hover:text-slate-400 hover:border-white/20 transition-all"
                  @dragover.prevent="handleDragOver($event, 'l2', category)"
                  @dragleave="handleDragLeave($event, 'l2')"
                  @drop="handleDrop($event, 'l2', category)"
                >
                  <Icon icon="lucide:plus" class="text-[10px]"></Icon>
                  <span class="text-[10px] font-bold">子字段</span>
                </div>
                <!-- Fields -->
                <div 
                  v-for="(subField, index) in subFields" 
                  :key="index"
                  draggable="true"
                  class="p-2.5 rounded-xl bg-[#1E2228] border border-white/5 flex items-center justify-between group hover:border-blue-500/30 hover:bg-white/5 transition-all cursor-grab"
                  @dragstart="handleDragStart($event, subField)"
                  @dragend="handleDragEnd"
                >
                  <span class="text-[12px] text-slate-300">{{ subField }}</span>
                  <Icon icon="lucide:grip-horizontal" class="text-slate-700 opacity-0 group-hover:opacity-100 transition-opacity text-xs"></Icon>
                </div>
              </div>
            </div>
            
            <!-- Empty Drop Zone Placeholder -->
            <div class="w-8 min-w-[12rem] flex flex-col items-center justify-center opacity-30 p-6 text-center">
              <div class="w-10 h-10 rounded-xl border border-dashed border-white/10 flex items-center justify-center mb-3">
                <Icon icon="lucide:layout-template" class="text-lg text-slate-600"></Icon>
              </div>
              <p class="text-[10px] font-medium text-slate-500 uppercase">可添加列</p>
            </div>
          </div>
        </section>
      </div>
      
      <!-- Modal Footer -->
      <div class="px-6 py-4 border-t border-white/5 flex items-center justify-between bg-white/[0.01]">
        <div class="flex items-center gap-2 text-[10px] text-slate-500 italic">
          <Icon icon="lucide:info" class="text-blue-500 text-xs"></Icon>
          <span>拖动字段从左上角创建主分类，到右侧列添加子栏目。</span>
        </div>
        <div class="flex gap-4">
          <button 
            class="px-6 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white font-medium transition-all"
            id="modal-reset-btn"
            @click="handleReset"
          >
            取消
          </button>
          <button 
            class="px-6 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            id="modal-save-btn"
            @click="handleSave"
            :disabled="localFieldList.length > 0"
          >
            确定
          </button>
        </div>
      </div>
    </div>
    

  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { Icon } from '@iconify/vue';

// 定义组件属性
const props = defineProps<{
  show: boolean;
  fieldList: string[];
  pbMap: Record<string, string[]>;
}>();

// 定义组件事件
const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'save', fieldMap: Record<string, string[]>): void;
}>();

// 响应式数据
const draggedField = ref<string | null>(null);

// 本地状态
const localFieldList = ref<string[]>([...props.fieldList]);
const localPbMap = ref<Record<string, string[]>>({ ...props.pbMap });

// 方法
const handleDragStart = (event: DragEvent, field: string) => {
  if (event.target) {
    (event.target as HTMLElement).classList.add('opacity-50');
  }
  draggedField.value = field;
};

const handleDragEnd = (event: DragEvent) => {
  if (event.target) {
    (event.target as HTMLElement).classList.remove('opacity-50');
  }
  draggedField.value = null;
  // 移除所有drop zone的active状态
  const l1DropZone = document.getElementById('l1-drop-zone');
  const l2DropZones = document.querySelectorAll('.l2-drop-zone');
  if (l1DropZone) {
    l1DropZone.classList.remove('drop-zone-active');
  }
  l2DropZones.forEach(zone => zone.classList.remove('drop-zone-active'));
};

const handleDragOver = (event: DragEvent, type: string, category?: string) => {
  event.preventDefault();
  if (type === 'l1') {
    const l1DropZone = document.getElementById('l1-drop-zone');
    if (l1DropZone) {
      l1DropZone.classList.add('drop-zone-active');
    }
  } else if (type === 'l2' && event.target) {
    (event.target as HTMLElement).classList.add('drop-zone-active');
  }
};

const handleDragLeave = (event: DragEvent, type: string) => {
  if (type === 'l1') {
    const l1DropZone = document.getElementById('l1-drop-zone');
    if (l1DropZone) {
      l1DropZone.classList.remove('drop-zone-active');
    }
  } else if (type === 'l2' && event.target) {
    (event.target as HTMLElement).classList.remove('drop-zone-active');
  }
};

const handleDrop = (event: DragEvent, type: string, category?: string) => {
  event.preventDefault();
  
  if (!draggedField.value) return;
  
  // 先从原位置移除字段
  removeFieldFromCurrentLocation(draggedField.value);
  
  if (type === 'l1') {
    // 添加为一级字段
    if (!localPbMap.value[draggedField.value]) {
      localPbMap.value[draggedField.value] = [];
    }
  } else if (type === 'l2' && category) {
    // 添加为二级子字段
    if (!localPbMap.value[category]) {
      localPbMap.value[category] = [];
    }
    // 检查是否已存在
    if (!localPbMap.value[category].includes(draggedField.value)) {
      localPbMap.value[category].push(draggedField.value);
    }
  }
  
  // 移除active状态
  const l1DropZone = document.getElementById('l1-drop-zone');
  const l2DropZones = document.querySelectorAll('.l2-drop-zone');
  if (l1DropZone) {
    l1DropZone.classList.remove('drop-zone-active');
  }
  l2DropZones.forEach(zone => zone.classList.remove('drop-zone-active'));
};

const removeFieldFromCurrentLocation = (field: string) => {
  // 从localFieldList中移除
  const fieldIndex = localFieldList.value.indexOf(field);
  if (fieldIndex > -1) {
    localFieldList.value.splice(fieldIndex, 1);
  }
  
  // 从所有分类中移除
  for (const cat in localPbMap.value) {
    const subFieldIndex = localPbMap.value[cat].indexOf(field);
    if (subFieldIndex > -1) {
      localPbMap.value[cat].splice(subFieldIndex, 1);
      break;
    }
  }
};

const removeCategory = (category: string) => {
  // 直接执行删除操作
  // 将该分类下的所有子字段移回未分类字段
  if (localPbMap.value[category]) {
    // 先将分类名称本身移回未分类字段
    localFieldList.value.push(category);
    // 再将子字段移回未分类字段
    localFieldList.value.push(...localPbMap.value[category]);
    // 删除分类
    delete localPbMap.value[category];
    
    // 提供视觉反馈 - 添加动画效果
    const categoryElement = document.querySelector(`[key="${category}"]`);
    if (categoryElement) {
      categoryElement.classList.add('animate-fade-out');
      setTimeout(() => {
        categoryElement.remove();
      }, 300);
    }
  }
};

const handleReset = () => {
  // 关闭弹窗
  emit('close');
};

const handleSave = () => {
  // 生成一级元素及其子字段的映射
  const fieldMap = { ...localPbMap.value };
  
  // 打印到控制台
  console.log('=== 字段配置映射 ===');
  console.log('生成的字段映射:', fieldMap);
  console.log('\n可读格式:');
  Object.entries(fieldMap).forEach(([category, fields]) => {
    console.log(`- ${category}: ${fields.join(', ')}`);
  });
  
  // 保存后关闭弹窗
  emit('save', fieldMap);
  emit('close');
};
</script>

<style scoped>
@import url('https://api.fontshare.com/v2/css?f[]=satoshi@700,500,400&f[]=cabinet-grotesk@800&display=swap');

:root {
  --bg-deep: #0F1115;
  --bg-sidebar: #16191E;
  --bg-card: #1E2228;
  --accent-blue: #3B82F6;
  --accent-teal: #14B8A6;
  --text-muted: #94A3B8;
}

.font-display {
  font-family: 'Cabinet Grotesk', sans-serif;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
  height: 4px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #334155;
  border-radius: 10px;
}

.glass-input {
  background-color: rgba(255,255,255,0.02);
  border: 1px solid rgba(255,255,255,0.08);
  transition: all 0.2s ease;
}

.glass-card {
  background: rgba(22,25,30,0.4);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255,255,255,0.05);
}

.drag-item:active {
  cursor: grabbing;
  transform: scale(0.98);
  opacity: 0.7;
}

.drop-zone-active {
  border-color: var(--accent-blue);
  background-color: rgba(59,130,246,0.05);
}

@keyframes pulse-subtle {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

@keyframes fade-in {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes fade-out {
  from {
    opacity: 1;
    transform: scale(1);
  }
  to {
    opacity: 0;
    transform: scale(0.95);
  }
}

.anim-pulse {
  animation: pulse-subtle 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.animate-fade-in {
  animation: fade-in 0.3s ease-out forwards;
}

.animate-fade-out {
  animation: fade-out 0.3s ease-in forwards;
}
</style>