<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { Search, RefreshCw, Trash2 } from 'lucide-vue-next';
import { ElDatePicker, ElMessage } from 'element-plus';
import * as wuyinApi from '../../api/wuyin';
import type { WuyinOrder } from '../../types';

// 吴饮订单详情页：左侧标签筛选栏（含各标签订单数），右侧分页表格。
// 查询走后端 /api/wuyin/query（tags 多选 + 关键词 + 时间范围）。

const props = defineProps<{
  tags: wuyinApi.WuyinTagConfig[];
}>();
const emit = defineEmits<{
  (e: 'status', msg: string): void;
}>();

interface TagStatRow {
  tag: string;
  count: number;
  selected: boolean;
}

const tagStats = ref<TagStatRow[]>([]);
const keyword = ref('');
const dateRange = ref<[Date, Date] | null>(null);
const isLoading = ref(false);
const orders = ref<WuyinOrder[]>([]);
const total = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);

const allTagNames = computed(() => {
  const names = props.tags.map(t => t.name);
  if (!names.includes('未分配')) names.unshift('未分配');
  return names;
});

const refreshTagStats = async () => {
  try {
    const stats = await wuyinApi.getTagStats(
      dateRange.value ? dateRange.value[0].toISOString() : undefined,
      dateRange.value ? dateRange.value[1].toISOString() : undefined,
    );
    const byTag = new Map((stats ?? []).map(s => [s.tag, s.count]));
    tagStats.value = allTagNames.value.map(name => ({
      tag: name,
      count: byTag.get(name) ?? 0,
      selected: tagStats.value.find(t => t.tag === name)?.selected ?? false,
    }));
  } catch {
    tagStats.value = allTagNames.value.map(name => ({ tag: name, count: 0, selected: false }));
  }
};

const selectedTags = computed(() => tagStats.value.filter(t => t.selected).map(t => t.tag));

const toggleTag = async (tag: string) => {
  const row = tagStats.value.find(t => t.tag === tag);
  if (row) row.selected = !row.selected;
  currentPage.value = 1;
  await query();
};

const query = async () => {
  isLoading.value = true;
  try {
    const params: wuyinApi.WuyinOrderQueryParams = {
      page: currentPage.value,
      limit: pageSize.value,
    };
    if (keyword.value.trim()) params.key = keyword.value.trim();
    if (selectedTags.value.length > 0) params.tags = selectedTags.value;
    if (dateRange.value) {
      params.startTime = dateRange.value[0].toISOString();
      params.endTime = dateRange.value[1].toISOString();
    }
    const resp = await wuyinApi.queryOrders(params);
    total.value = resp.count;
    orders.value = (resp.list ?? []).map(item => ({
      id: String(item.id ?? item.ID ?? 0),
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
    }));
  } catch (error) {
    ElMessage.error((error as Error).message || '查询失败');
  } finally {
    isLoading.value = false;
  }
};

const changeRowTag = async (order: WuyinOrder, tag: string) => {
  try {
    await wuyinApi.updateOrder(order.dbId, { 标签: tag });
    order.tag = tag;
  } catch (error) {
    ElMessage.error((error as Error).message || '标签修改失败');
  }
};

const removeOrder = async (order: WuyinOrder) => {
  try {
    await wuyinApi.deleteOrder(order.dbId);
    ElMessage.success('已删除');
    await Promise.all([query(), refreshTagStats()]);
  } catch (error) {
    ElMessage.error((error as Error).message || '删除失败');
  }
};

const totalPages = computed(() => Math.ceil(total.value / pageSize.value) || 1);

// 模板不能引用全局 Promise，查询入口统一走这里
const refreshAll = async () => {
  currentPage.value = 1;
  await Promise.all([refreshTagStats(), query()]);
};

onMounted(async () => {
  await refreshTagStats();
  await query();
});
</script>

<template>
  <div class="flex gap-6">
    <!-- 标签筛选侧栏 -->
    <aside class="w-52 shrink-0 space-y-3">
      <div class="settings-card rounded-3xl p-5 space-y-2">
        <p class="text-xs font-bold uppercase tracking-widest text-[var(--text-faint)] px-1">仓库标签</p>
        <button
          v-for="t in tagStats"
          :key="t.tag"
          @click="toggleTag(t.tag)"
          class="w-full flex items-center justify-between px-3 py-2 rounded-xl text-sm transition-all"
          :class="t.selected
            ? 'bg-[var(--accent-soft-bg)] text-[var(--accent)] font-bold border-r-[3px] border-[var(--accent)]'
            : 'text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'"
        >
          <span class="truncate">{{ t.tag }}</span>
          <span class="text-xs px-2 py-0.5 rounded-full bg-[var(--bg-hover-strong)]">{{ t.count }}</span>
        </button>
        <p class="text-[10px] text-[var(--text-faint)] px-1 pt-1">点击标签筛选，可多选</p>
      </div>
    </aside>

    <!-- 主区 -->
    <section class="flex-1 min-w-0 space-y-4">
      <div class="flex items-center justify-between gap-4 flex-wrap">
        <h2 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">吴饮订单详情 <span class="text-sm font-normal text-[var(--text-muted)]">共 {{ total }} 条</span></h2>
        <div class="flex items-center gap-3">
          <ElDatePicker
            v-model="dateRange"
            type="datetimerange"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            class="!w-72"
            :clearable="true"
          />
          <div class="relative">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-faint)] w-4 h-4" />
            <input v-model="keyword" @keyup.enter="refreshAll" placeholder="关键词（收件人/电话/地址）" class="glass-input pl-10 pr-4 py-2.5 rounded-xl text-sm w-64" />
          </div>
          <button
            @click="refreshAll"
            class="px-4 py-2.5 rounded-xl bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-sm flex items-center gap-2"
          >
            <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': isLoading }" /> 查询
          </button>
        </div>
      </div>

      <div class="settings-card rounded-3xl overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-[var(--text-faint)] border-b border-[var(--border)]">
                <th class="px-4 py-3 font-semibold">ID</th>
                <th class="px-4 py-3 font-semibold">标签</th>
                <th class="px-4 py-3 font-semibold">收件人</th>
                <th class="px-4 py-3 font-semibold">手机号</th>
                <th class="px-4 py-3 font-semibold">收货地址</th>
                <th class="px-4 py-3 font-semibold">商品规格</th>
                <th class="px-4 py-3 font-semibold">发送人</th>
                <th class="px-4 py-3 font-semibold">入库时间</th>
                <th class="px-4 py-3 font-semibold text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="orders.length === 0">
                <td colspan="9" class="px-4 py-12 text-center text-[var(--text-faint)]">{{ isLoading ? '查询中...' : '没有符合条件的订单' }}</td>
              </tr>
              <tr v-for="order in orders" :key="order.id" class="border-b border-[var(--border)] last:border-0 hover:bg-[var(--bg-hover)] transition-colors">
                <td class="px-4 py-3 text-[var(--text-faint)]">{{ order.dbId }}</td>
                <td class="px-4 py-3">
                  <select
                    :value="order.tag"
                    @change="changeRowTag(order, ($event.target as HTMLSelectElement).value)"
                    class="bg-[var(--accent-soft-bg)] text-[var(--accent-soft)] text-xs font-medium px-2.5 py-1 rounded-full border-0 cursor-pointer outline-none"
                  >
                    <option v-for="t in allTagNames" :key="t" :value="t">{{ t }}</option>
                    <option v-if="!allTagNames.includes(order.tag)" :value="order.tag">{{ order.tag }}</option>
                  </select>
                </td>
                <td class="px-4 py-3 text-[var(--text-primary)] font-medium">{{ order.name || '—' }}</td>
                <td class="px-4 py-3 text-[var(--text-secondary)] font-mono text-xs">{{ order.phone || '—' }}</td>
                <td class="px-4 py-3 text-[var(--text-secondary)] max-w-xs truncate" :title="order.address">{{ order.address }}</td>
                <td class="px-4 py-3 text-[var(--text-secondary)] font-mono text-xs whitespace-nowrap">{{ order.items }}</td>
                <td class="px-4 py-3 text-[var(--text-muted)] text-xs max-w-[10rem] truncate" :title="order.sender">{{ order.sender || '—' }}</td>
                <td class="px-4 py-3 text-[var(--text-muted)] text-xs whitespace-nowrap">{{ order.createdAt ? order.createdAt.replace('T', ' ').slice(0, 16) : '—' }}</td>
                <td class="px-4 py-3 text-right">
                  <button @click="removeOrder(order)" class="px-3 py-1.5 rounded-lg bg-[var(--bg-hover)] hover:bg-red-500/10 hover:text-red-400 text-[var(--text-muted)] text-xs inline-flex items-center gap-1">
                    <Trash2 class="w-3 h-3" /> 删除
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t border-[var(--border)] text-xs text-[var(--text-muted)]">
          <span>第 {{ currentPage }} / {{ totalPages }} 页 · 共 {{ total }} 条</span>
          <div class="space-x-2">
            <button @click="() => { currentPage = Math.max(1, currentPage - 1); query(); }" :disabled="currentPage <= 1" class="px-3 py-1 rounded-lg bg-[var(--bg-hover)] disabled:opacity-40">上一页</button>
            <button @click="() => { currentPage = Math.min(totalPages, currentPage + 1); query(); }" :disabled="currentPage >= totalPages" class="px-3 py-1 rounded-lg bg-[var(--bg-hover)] disabled:opacity-40">下一页</button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
