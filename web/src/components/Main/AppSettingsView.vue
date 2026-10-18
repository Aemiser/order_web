<script setup lang="ts">
import { 
  FolderOpen, 
  Terminal, 
  Edit3, 
  Cpu, 
  Info 
} from 'lucide-vue-next';
import { ref } from 'vue';

const exportPath = ref('C:\\Users\\Admin\\Documents\\Orders\\Exports');
const selectedModel = ref('智能通用模型 (推荐)');
const themeMode = ref('dark');

const saveSettings = () => {
  // Logic to save settings
  alert('设置已保存');
};
</script>

<template>
  <div class="space-y-10 pb-16">
    <div class="flex flex-col gap-2">
      <h2 class="text-3xl font-bold tracking-tight font-display text-[var(--text-primary)]">应用设置</h2>
      <p class="text-[var(--text-muted)]">管理您的订单解析偏好、存储路径及系统配置</p>
    </div>

    <!-- Export Path Section -->
    <section class="p-8 space-y-6 settings-card rounded-3xl">
      <div class="flex items-start gap-4">
        <div class="flex items-center justify-center text-2xl text-[var(--accent)] w-12 h-12 rounded-2xl bg-[var(--accent-soft-bg)]">
          <FolderOpen class="w-6 h-6" />
        </div>
        <div class="flex-1">
          <h3 class="text-xl font-bold font-display">导出路径设置</h3>
          <p class="text-sm text-[var(--text-faint)] mt-1">所有生成的 XLSX 表格将自动保存到此文件夹，请确保程序具有写入权限。</p>
        </div>
      </div>
      <div class="flex gap-3 mt-4">
        <div class="relative flex-1">
          <Terminal class="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--text-faint)] w-4 h-4" />
          <input 
            type="text" 
            v-model="exportPath" 
            readonly 
            class="w-full pl-12 pr-4 py-4 rounded-2xl glass-input text-sm text-[var(--text-secondary)] font-mono"
          >
        </div>
        <button class="px-6 py-4 rounded-2xl bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-primary)] font-medium border border-[var(--border-strong)] transition-all flex items-center gap-2 active:scale-95">
          <Edit3 class="w-4 h-4" /> 修改路径
        </button>
      </div>
    </section>

<!--    &lt;!&ndash; AI Engine Section &ndash;&gt;-->
<!--    <section class="p-8 space-y-8 settings-card rounded-3xl">-->
<!--      <div class="flex items-start gap-4">-->
<!--        <div class="flex items-center justify-center text-2xl text-[var(--teal)] w-12 h-12 rounded-2xl bg-teal-600/10">-->
<!--          <Cpu class="w-6 h-6" />-->
<!--        </div>-->
<!--        <div class="flex-1">-->
<!--          <h3 class="text-xl font-bold font-display">AI 识别引擎</h3>-->
<!--          <p class="text-sm text-[var(--text-faint)] mt-1">根据您的订单复杂度选择最合适的解析模型。</p>-->
<!--        </div>-->
<!--      </div>-->
<!--      <div class="space-y-3">-->
<!--        <label class="text-xs font-bold text-[var(--text-faint)] uppercase tracking-widest">智能模型选择</label>-->
<!--        <select -->
<!--          v-model="selectedModel"-->
<!--          class="w-full px-5 py-4 rounded-2xl glass-input text-sm text-[var(--text-secondary)] bg-[#1e2228] outline-none border border-[var(--border-strong)] focus:border-[var(--accent)]"-->
<!--        >-->
<!--          <option>智能通用模型 (推荐)</option>-->
<!--          <option>电商专用模型</option>-->
<!--          <option>极速轻量模型</option>-->
<!--        </select>-->
<!--      </div>-->
<!--    </section>-->

    <!-- Preferences Section -->
<!--    <section class="p-8 settings-card rounded-3xl">-->
<!--      <div class="grid grid-cols-1 md:grid-cols-2 gap-12">-->
<!--        &lt;!&ndash; Format Preference &ndash;&gt;-->
<!--        <div class="space-y-6">-->
<!--          <div>-->
<!--            <h4 class="text-lg font-bold font-display">导出格式偏好</h4>-->
<!--            <p class="text-xs text-[var(--text-faint)] mt-1">选择默认生成的文档格式类型</p>-->
<!--          </div>-->
<!--          <div class="grid grid-cols-2 gap-4">-->
<!--            <div class="p-5 rounded-2xl border-2 border-blue-600 bg-[var(--accent)]/5 flex flex-col items-center gap-3 cursor-pointer">-->
<!--              <div class="w-10 h-10 flex items-center justify-center bg-green-600/20 rounded-lg">-->
<!--                <span class="text-green-500 font-bold text-xs">XLSX</span>-->
<!--              </div>-->
<!--              <span class="text-sm font-bold">XLSX</span>-->
<!--            </div>-->
<!--            <div class="p-5 rounded-2xl border border-[var(--border)] bg-[var(--bg-hover-faint)] opacity-40 flex flex-col items-center gap-3 cursor-not-allowed grayscale">-->
<!--              <div class="w-10 h-10 flex items-center justify-center bg-slate-600/20 rounded-lg">-->
<!--                <span class="text-[var(--text-faint)] font-bold text-xs">CSV</span>-->
<!--              </div>-->
<!--              <span class="text-sm font-medium">CSV</span>-->
<!--            </div>-->
<!--          </div>-->
<!--        </div>-->

<!--        &lt;!&ndash; Theme Mode &ndash;&gt;-->
<!--        <div class="space-y-6">-->
<!--          <div>-->
<!--            <h4 class="text-lg font-bold font-display">主题界面模式</h4>-->
<!--            <p class="text-xs text-[var(--text-faint)] mt-1">根据环境光线自动调节或手动锁定主题</p>-->
<!--          </div>-->
<!--          <div class="flex p-1.5 rounded-2xl glass-input bg-[var(--bg-hover)]">-->
<!--            <button -->
<!--              @click="themeMode = 'light'"-->
<!--              class="flex-1 py-3 rounded-xl text-xs font-bold transition-all"-->
<!--              :class="themeMode === 'light' ? 'bg-[var(--accent)] text-[var(--text-on-accent)] shadow-lg shadow-[var(--accent-glow)]' : 'text-[var(--text-muted)] hover:text-[var(--text-on-accent)]'"-->
<!--            >-->
<!--              浅色模式-->
<!--            </button>-->
<!--            <button -->
<!--              @click="themeMode = 'dark'"-->
<!--              class="flex-1 py-3 rounded-xl text-xs font-bold transition-all"-->
<!--              :class="themeMode === 'dark' ? 'bg-[var(--accent)] text-[var(--text-on-accent)] shadow-lg shadow-[var(--accent-glow)]' : 'text-[var(--text-muted)] hover:text-[var(--text-on-accent)]'"-->
<!--            >-->
<!--              深色模式-->
<!--            </button>-->
<!--            <button -->
<!--              @click="themeMode = 'system'"-->
<!--              class="flex-1 py-3 rounded-xl text-xs font-bold transition-all"-->
<!--              :class="themeMode === 'system' ? 'bg-[var(--accent)] text-[var(--text-on-accent)] shadow-lg shadow-[var(--accent-glow)]' : 'text-[var(--text-muted)] hover:text-[var(--text-on-accent)]'"-->
<!--            >-->
<!--              跟随系统-->
<!--            </button>-->
<!--          </div>-->
<!--        </div>-->
<!--      </div>-->
<!--    </section>-->

    <!-- Footer Actions -->
    <div class="pt-6 border-t border-[var(--border)] flex items-center justify-between">
      <div class="flex items-center gap-2 text-[var(--text-faint)] text-xs italic">
        <Info class="w-4 h-4 text-[var(--accent)]" /> 最后保存时间: 2023-11-21 17:45:22
      </div>
      <button 
        @click="saveSettings"
        class="px-12 py-4 rounded-2xl bg-[var(--accent)] hover:bg-[var(--accent)] text-[var(--text-on-accent)] font-bold text-lg shadow-2xl shadow-[var(--accent-glow)] transition-all active:scale-95"
      >
        保存所有配置
      </button>
    </div>
  </div>
</template>

<style scoped>
.font-display {
  font-family: 'Cabinet Grotesk', sans-serif;
}
.glass-input {
  background-color: var(--bg-input);
  border: 1px solid var(--border-strong);
  transition: all 0.2s ease;
}
.glass-input:focus {
  outline: none;
  border-color: var(--accent);
  background-color: var(--bg-hover);
}
.settings-card {
  background: var(--bg-card);
  backdrop-filter: blur(10px);
  border: 1px solid var(--border);
  transition: all 0.3s ease;
}
.settings-card:hover {
  border-color: var(--accent-border);
}
</style>
