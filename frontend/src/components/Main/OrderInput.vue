<script setup lang="ts">
import { ref } from 'vue';
import { Sparkles, Loader2 } from 'lucide-vue-next';

defineProps<{
  inputText: string;
  isAnalyzing: boolean;
}>();

// 模式选择：false 为"重来"，true 为"追加"
const isAppendMode = ref(false);

defineEmits<{
  (e: 'update:inputText', value: string): void;
  (e: 'clear'): void;
  (e: 'analyze', isAppend: boolean): void;
}>();
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-bold tracking-tight">新建订单解析</h2>
      <span class="text-sm text-slate-500">支持直接粘贴文本或拖入文件</span>
    </div>
    <div class="relative group">
      <textarea 
        :value="inputText"
        @input="$emit('update:inputText', ($event.target as HTMLTextAreaElement).value)"
        placeholder="请在此处输入或粘贴原始订单详情...&#10;示例：王小二 13800138000 上海市静安区XX路XX号 苹果x1 香蕉x2" 
        class="w-full h-48 p-6 rounded-2xl bg-white/5 border border-white/10 focus:border-blue-500 focus:bg-white/[0.07] outline-none text-slate-200 placeholder:text-slate-600 resize-none font-mono text-sm leading-relaxed transition-all"
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
            <div class="w-9 h-5 bg-white/10 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
            <span class="ml-2 text-sm font-medium text-slate-300">
              {{ isAppendMode ? '追加' : '重来' }}
            </span>
          </label>
        </div>
        <button 
          @click="$emit('clear')"
          class="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 text-sm font-medium transition-colors"
        >
          清空输入
        </button>
        <button 
          @click="$emit('analyze', isAppendMode)"
          :disabled="isAnalyzing || !inputText.trim()"
          class="px-6 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-bold flex items-center gap-2 shadow-lg shadow-blue-600/20 transition-all"
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
