<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { Sparkles, Loader2, ChevronDown, ChevronUp } from 'lucide-vue-next';
import * as systemApi from '../../api/system';

defineProps<{
  inputText: string;
  isAnalyzing: boolean;
}>();

// 模式选择：false 为"重来"，true 为"追加"
const isAppendMode = ref(false);

// AI模型选择
const aiModels = ref<Map<string, number>>(new Map());
const selectedModel = ref<string>('');
const isDropdownOpen = ref(false);
const isLoadingModels = ref(false);
const modelError = ref('');

defineEmits<{
  (e: 'update:inputText', value: string): void;
  (e: 'clear'): void;
  (e: 'analyze', isAppend: boolean, modelId: number): void;
}>();

// 获取AI模型列表
const fetchAIModels = async () => {
  isLoadingModels.value = true;
  modelError.value = '';
  try {
    const models = await systemApi.getAIModels();
    aiModels.value = new Map(Object.entries(models));
    // 默认选择第一个模型
    if (aiModels.value.size > 0) {
      selectedModel.value = Array.from(aiModels.value.keys())[0];
    }
  } catch (error) {
    console.error('获取AI模型失败:', error);
    modelError.value = '获取模型列表失败，请重试';
  } finally {
    isLoadingModels.value = false;
  }
};

// 切换下拉菜单
const toggleDropdown = () => {
  isDropdownOpen.value = !isDropdownOpen.value;
};

// 选择模型
const selectModel = (model: string) => {
  selectedModel.value = model;
  isDropdownOpen.value = false;
};

// 处理点击外部关闭下拉菜单
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement;
  if (!target.closest('.ai-model-selector')) {
    isDropdownOpen.value = false;
  }
};

// 生命周期钩子：组件挂载时获取AI模型
onMounted(() => {
  fetchAIModels();
  document.addEventListener('click', handleClickOutside);
});
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-bold tracking-tight">新建订单解析</h2>
      <span class="text-sm text-[var(--text-faint)]">支持直接粘贴文本或拖入文件</span>
    </div>
    <div class="relative group">
      <textarea 
        :value="inputText"
        @input="$emit('update:inputText', ($event.target as HTMLTextAreaElement).value)"
        placeholder="请在此处输入或粘贴原始订单详情...&#10;示例：王小二 13800138000 上海市静安区XX路XX号 苹果x1 香蕉x2" 
        class="w-full h-48 p-6 rounded-2xl bg-[var(--bg-hover)] border border-[var(--border-strong)] focus:border-[var(--accent)] focus:bg-white/[0.07] outline-none text-[var(--text-secondary)] placeholder:text-[var(--text-faint)] resize-none font-mono text-sm leading-relaxed transition-all"
      ></textarea>
      <div class="absolute bottom-4 right-4 flex items-center gap-3">
        <!-- 模式选择复选框 -->
        <div class="flex items-center gap-2">
          <label class="relative inline-flex items-center cursor-pointer">
            <input 
              type="checkbox" 
              v-model="isAppendMode" 
              class="sr-only peer"
            />
            <div class="w-9 h-5 bg-[var(--bg-hover-strong)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-[var(--text-on-accent)] after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-[var(--text-primary)] after:border-[var(--text-primary)] after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[var(--accent)]"></div>
            <span class="ml-2 text-sm font-medium text-[var(--text-secondary)]">
              {{ isAppendMode ? '追加' : '重来' }}
            </span>
          </label>
        </div>
        <button 
          @click="$emit('clear')"
          class="px-4 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] text-sm font-medium transition-colors"
        >
          清空输入
        </button>
        <!-- AI模型选择器 -->
        <div class="ai-model-selector relative">
          <div 
            @click="toggleDropdown"
            class="flex items-center gap-2 px-4 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-sm font-medium cursor-pointer transition-colors"
          >
            <span v-if="isLoadingModels">加载中...</span>
            <span v-else-if="selectedModel">AI模型：{{ selectedModel }}</span>
            <span v-else>选择模型</span>
            <ChevronDown v-if="!isDropdownOpen" class="w-4 h-4" />
            <ChevronUp v-else class="w-4 h-4" />
          </div>
          
          <!-- 下拉菜单 -->
          <div 
            v-if="isDropdownOpen && aiModels.size > 0"
            class="absolute top-full mt-2 w-48 bg-[var(--bg-card)] border border-[var(--border-strong)] rounded-lg shadow-lg z-10"
          >
            <div 
              v-for="model in Array.from(aiModels.keys())" 
              :key="model"
              @click="selectModel(model)"
              class="px-4 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] cursor-pointer transition-colors"
              :class="{ 'bg-[var(--accent-soft-bg-strong)] text-[var(--accent-soft)]': model === selectedModel }"
            >
              {{ model }}
            </div>
          </div>
          
          <!-- 错误提示 -->
          <div v-if="modelError" class="absolute top-full mt-2 w-48 text-xs text-red-400">
            {{ modelError }}
          </div>
        </div>
        <button 
          @click="$emit('analyze', isAppendMode, aiModels.get(selectedModel) || 0)"
          :disabled="isAnalyzing || !inputText.trim() || !selectedModel"
          class="px-6 py-2 rounded-lg bg-[var(--accent)] hover:bg-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed text-[var(--text-on-accent)] text-sm font-bold flex items-center gap-2 shadow-lg shadow-[var(--accent-glow)] transition-all"
        >
          <template v-if="isAnalyzing">
            <Loader2 class="w-4 h-4 animate-spin" />
            正在识别...
          </template>
          <template v-else>
            识别数据 <Sparkles class="w-4 h-4" />
          </template>
        </button>
      </div>
    </div>
  </section>
</template>
