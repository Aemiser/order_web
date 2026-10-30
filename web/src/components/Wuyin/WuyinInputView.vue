<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { Sparkles, Loader2, Upload, Info, Download } from 'lucide-vue-next';
import { ElDatePicker, ElMessage } from 'element-plus';
import * as systemApi from '../../api/system';
import * as wuyinApi from '../../api/wuyin';
import WuyinPreviewTable from './WuyinPreviewTable.vue';
import { BASE } from '../../api/request';
import type { WuyinOrder } from '../../types';

// 吴饮订单输入页：文本识别（同步接口，与一点点当前用法一致）+ 表格导入 + 预览。
// 自持全部状态（不上升 App.vue），保持一点点链路零改动。

const props = defineProps<{
  tags: string[];
}>();
const emit = defineEmits<{
  (e: 'status', msg: string): void;
}>();

const inputText = ref('');
const isAnalyzing = ref(false);
const isAppendMode = ref(false);
const orders = ref<WuyinOrder[]>([]);

// AI 模型选择
const aiModels = ref<Map<string, number>>(new Map());
const selectedModel = ref('');
const isDropdownOpen = ref(false);
const toggleDropdown = () => (isDropdownOpen.value = !isDropdownOpen.value);
const selectModel = (m: string) => {
  selectedModel.value = m;
  isDropdownOpen.value = false;
};
const handleClickOutside = (event: MouseEvent) => {
  if (!(event.target as HTMLElement).closest('.ai-model-selector')) isDropdownOpen.value = false;
};

// 表格导入
const fileInput = ref<HTMLInputElement | null>(null);
const importTag = ref('');
const isImporting = ref(false);
const triggerImport = () => fileInput.value?.click();
const handleImportFile = async (e: Event) => {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  isImporting.value = true;
  try {
    const list = await wuyinApi.importExcel(file, importTag.value);
    orders.value = toOrders(list);
    ElMessage.success(`导入成功，预览共 ${orders.value.length} 条`);
    emit('status', `表格导入 ${file.name} 成功`);
  } catch (error) {
    ElMessage.error((error as Error).message || '导入失败');
  } finally {
    isImporting.value = false;
    input.value = ''; // 允许重复选择同一文件
  }
};

// ---- 数据映射 ----
const toOrder = (item: wuyinApi.WuyinAddressInfo): WuyinOrder => ({
  id: String(item.VID ?? item.PreviewID ?? item.id ?? 0),
  dbId: item.id ?? item.ID ?? 0,
  name: item.收件人 || item.Recipient || '',
  phone: item.手机号 || item.Phone || '',
  address: item.收货地址 || item.Address || '',
  province: item.省份 || item.Province || '',
  items: item.商品规格 || item.ProductSpecs || '-',
  note: item.备注 || item.Remarks || '',
  tag: item.标签 || item.Tag || '未分配',
  sender: item.发送人 || item.Sender || '',
  createdAt: item.createdAt,
});
const toOrders = (list: wuyinApi.WuyinAddressInfo[] | null | undefined): WuyinOrder[] =>
  (list ?? []).map(toOrder);

// ---- 识别 ----
const analyze = async () => {
  if (!inputText.value.trim() || !selectedModel.value) return;
  isAnalyzing.value = true;
  emit('status', '正在识别吴饮订单...');
  try {
    const result = await wuyinApi.recognition(
      inputText.value,
      aiModels.value.get(selectedModel.value) || 0,
      isAppendMode.value ? 1 : 0,
    );
    const mapped = toOrders(result);
    if (!isAppendMode.value) orders.value = [];
    orders.value = [...orders.value, ...mapped];
    if (mapped.length > 0) {
      inputText.value = '';
      ElMessage.success(`识别到 ${mapped.length} 条订单`);
      emit('status', `识别成功 ${mapped.length} 条`);
    } else {
      ElMessage.warning('未识别到订单');
    }
  } catch (error) {
    ElMessage.error((error as Error).message || '识别失败');
  } finally {
    isAnalyzing.value = false;
  }
};

const refresh = async () => {
  try {
    orders.value = toOrders(await wuyinApi.getPreview());
  } catch {
    orders.value = [];
  }
};

const handleDelete = async (vid: number) => {
  try {
    await wuyinApi.deletePreview(vid);
    await refresh();
  } catch (error) {
    ElMessage.error((error as Error).message || '删除失败');
  }
};

// ---- 入库与导出 ----
const showConfirmModal = ref(false);
const selectedDateTime = ref<Date>(new Date());
const createOrder = () => {
  selectedDateTime.value = new Date();
  showConfirmModal.value = true;
};
const confirmCreate = async () => {
  showConfirmModal.value = false;
  try {
    await wuyinApi.createOrder(selectedDateTime.value);
    ElMessage.success('订单已入库');
    emit('status', '吴饮订单入库成功');
    await refresh();
  } catch (error) {
    ElMessage.error((error as Error).message || '入库失败');
  }
};
const isExporting = ref(false);
const exportExcel = async () => {
  isExporting.value = true;
  try {
    const resp = await wuyinApi.exportOrder();
    if (resp?.url) window.open(`${BASE}${resp.url}`, '_blank');
  } catch (error) {
    ElMessage.error((error as Error).message || '导出失败');
  } finally {
    isExporting.value = false;
  }
};

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
  void (async () => {
    try {
      const models = await systemApi.getAIModels();
      aiModels.value = new Map(Object.entries(models));
      if (aiModels.value.size > 0) selectedModel.value = Array.from(aiModels.value.keys())[0];
    } catch {
      /* 模型列表失败时按钮置灰即可 */
    }
  })();
  void refresh();
});
</script>

<template>
  <div class="space-y-6 pb-24">
    <!-- 输入区 -->
    <section class="space-y-4">
      <div class="flex items-baseline gap-3">
        <h2 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">吴饮订单解析</h2>
        <span class="text-sm text-[var(--text-muted)]">粘贴群聊记录，或导入仓库日报表格</span>
      </div>
      <div class="relative group">
        <textarea
          v-model="inputText"
          placeholder="粘贴吾饮良品群聊记录...&#10;示例：[2026-10-29 16:52] 广西～南昌订货微信: 广西百色市平果（潘雪晶）吾饮 18277682337 1件香水柠檬"
          class="w-full h-56 p-6 rounded-2xl bg-[var(--bg-hover)] border border-[var(--border-strong)] focus:border-[var(--accent)] focus:bg-[var(--bg-card)] outline-none text-[var(--text-secondary)] placeholder:text-[var(--text-faint)] resize-none font-mono text-sm leading-relaxed transition-all"
        ></textarea>
        <div class="absolute bottom-4 right-4 flex items-center gap-3">
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="isAppendMode" class="sr-only peer" />
            <div class="w-9 h-5 bg-[var(--bg-hover-strong)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-[var(--text-on-accent)] after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-[var(--text-primary)] after:border after:border-[var(--text-primary)] after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[var(--accent)]"></div>
            <span class="ml-2 text-sm font-medium text-[var(--text-secondary)]">{{ isAppendMode ? '追加' : '重来' }}</span>
          </label>

          <!-- 表格导入 -->
          <div class="flex items-center gap-2">
            <select v-model="importTag" class="px-3 py-2 rounded-lg bg-[var(--bg-hover)] border border-[var(--border-strong)] text-[var(--text-secondary)] text-sm outline-none">
              <option value="">导入标签：默认</option>
              <option v-for="t in props.tags" :key="t" :value="t">{{ t }}</option>
            </select>
            <input ref="fileInput" type="file" accept=".xlsx,.xls" class="hidden" @change="handleImportFile" />
            <button
              @click="triggerImport"
              :disabled="isImporting"
              class="px-4 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-sm font-medium transition-colors flex items-center gap-2 disabled:opacity-50"
            >
              <Loader2 v-if="isImporting" class="w-4 h-4 animate-spin" />
              <Upload v-else class="w-4 h-4" />
              导入表格
            </button>
          </div>

          <!-- AI 模型选择 -->
          <div class="ai-model-selector relative">
            <div @click="toggleDropdown" class="flex items-center gap-2 px-4 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-sm font-medium cursor-pointer transition-colors">
              <span v-if="selectedModel">AI模型：{{ selectedModel }}</span>
              <span v-else>选择模型</span>
            </div>
            <div v-if="isDropdownOpen && aiModels.size > 0" class="absolute top-full mt-2 w-48 bg-[var(--bg-card)] border border-[var(--border-strong)] rounded-lg shadow-lg z-10">
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
          </div>

          <button
            @click="analyze"
            :disabled="isAnalyzing || !inputText.trim() || !selectedModel"
            class="px-6 py-2 rounded-lg bg-[var(--accent)] hover:bg-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed text-[var(--text-on-accent)] text-sm font-bold flex items-center gap-2 shadow-lg shadow-[var(--accent-glow)] transition-all"
          >
            <template v-if="isAnalyzing">
              <Loader2 class="w-4 h-4 animate-spin" /> 正在识别...
            </template>
            <template v-else>
              识别数据 <Sparkles class="w-4 h-4" />
            </template>
          </button>
        </div>
      </div>
    </section>

    <!-- 预览表 -->
    <WuyinPreviewTable :orders="orders" :tags="props.tags" @delete="handleDelete" @refresh="refresh" />

    <!-- 底部操作 -->
    <footer class="fixed bottom-0 left-64 right-0 h-20 bg-[var(--bg-sidebar)] border-t border-[var(--border)] px-8 flex items-center justify-center z-20">
      <div class="w-full max-w-7xl mx-auto flex items-center justify-between gap-4">
        <div class="flex items-center gap-2 text-sm text-[var(--text-muted)]">
          <Info class="w-4 h-4 text-[var(--accent)]" />
          预览共 {{ orders.length }} 条订单（件计价）
        </div>
        <div class="flex items-center gap-4">
          <button @click="createOrder" class="px-8 py-3 rounded-xl bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-primary)] font-medium transition-all">
            确认数据无误
          </button>
          <button
            @click="exportExcel"
            :disabled="orders.length === 0 || isExporting"
            class="px-10 py-3 rounded-xl bg-[var(--accent)] hover:bg-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed text-[var(--text-on-accent)] font-bold flex items-center gap-2 shadow-xl shadow-[var(--accent-glow)] transition-all"
          >
            <Loader2 v-if="isExporting" class="w-4 h-4 animate-spin" />
            <Download v-else class="w-4 h-4" />
            导出 XLSX 表格
          </button>
        </div>
      </div>
    </footer>

    <!-- 入库确认 -->
    <div v-if="showConfirmModal" class="fixed inset-0 bg-[var(--overlay)] flex items-center justify-center z-50">
      <div class="bg-[var(--bg-sidebar)] border border-[var(--border-strong)] rounded-2xl p-8 max-w-md w-full mx-4">
        <h3 class="text-xl font-bold text-[var(--text-primary)] mb-4 font-display">确认操作</h3>
        <p class="text-[var(--text-secondary)] mb-6">请确保本批吴饮订单正确（标签将随订单入库），确认后影响结账！</p>
        <div class="mb-6">
          <label class="block text-sm font-medium text-[var(--text-muted)] mb-2">选择订单日期</label>
          <ElDatePicker v-model="selectedDateTime" type="datetime" placeholder="选择日期和时间" class="w-full" :clearable="false" size="large" :disabled-date="(time: any) => time.getTime() > new Date().getTime()" />
        </div>
        <div class="flex justify-end gap-4">
          <button @click="showConfirmModal = false" class="px-6 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] hover:text-[var(--text-primary)] font-medium transition-all">取消</button>
          <button @click="confirmCreate" class="px-6 py-2 rounded-lg bg-[var(--accent)] hover:bg-[var(--accent)] text-[var(--text-on-accent)] font-medium transition-all">确定</button>
        </div>
      </div>
    </div>
  </div>
</template>
