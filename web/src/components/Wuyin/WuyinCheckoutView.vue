<script setup lang="ts">
import { ref } from 'vue';
import { RefreshCw, DollarSign, Wallet, Pencil, X, Check } from 'lucide-vue-next';
import { ElDatePicker, ElMessage } from 'element-plus';
import * as wuyinApi from '../../api/wuyin';
import { BASE } from '../../api/request';

// 吴饮结账页：按标签分组的账单卡片。每个标签一张独立表（商品 × 日期 × 件），
// 单价统一编辑后保存，导出时每个标签一个 sheet；也可只导出选中标签。

interface CheckoutProduct {
  name: string;
  spec: string;
  itemsPerDate: number[];
  totalItems: number;
  pricePerItem: number;
  totalAmount: number;
}
interface CheckoutGroup {
  tag: string;
  dates: string[];
  products: CheckoutProduct[] | null;
  count: number;
  grandTotal: number;
}

const dateRange = ref<[Date, Date] | null>(null);
const groups = ref<CheckoutGroup[]>([]);
const isLoading = ref(false);
const isEditMode = ref(false);
const priceDraft = ref<Record<string, number>>({});

const query = async () => {
  if (!dateRange.value) {
    ElMessage.warning('请先选择时间范围');
    return;
  }
  isLoading.value = true;
  try {
    const resp = await wuyinApi.getDayData(dateRange.value[0].toISOString(), dateRange.value[1].toISOString());
    groups.value = (resp ?? []) as CheckoutGroup[];
    if (groups.value.length === 0) ElMessage.info('该区间没有订单数据');
  } catch (error) {
    ElMessage.error((error as Error).message || '查询失败');
  } finally {
    isLoading.value = false;
  }
};

const enterEditMode = () => {
  priceDraft.value = {};
  for (const g of groups.value) {
    for (const p of g.products ?? []) {
      priceDraft.value[`${g.tag}|${p.name}|${p.spec}`] = p.pricePerItem;
    }
  }
  isEditMode.value = true;
};

const savePrices = async () => {
  // 汇总所有标签下的价格（同名商品同规格在多个标签下取最后一次输入，口径统一）
  const byProduct = new Map<string, { productName: string; spec: string; pricePerItem: number }>();
  for (const g of groups.value) {
    for (const p of g.products ?? []) {
      const v = priceDraft.value[`${g.tag}|${p.name}|${p.spec}`];
      if (v !== undefined) byProduct.set(`${p.name}|${p.spec}`, { productName: p.name, spec: p.spec, pricePerItem: v });
    }
  }
  if (byProduct.size === 0) {
    ElMessage.warning('没有可保存的价格');
    return;
  }
  isLoading.value = true;
  try {
    const resp = await wuyinApi.setPrice(Array.from(byProduct.values()));
    groups.value = (resp ?? []) as CheckoutGroup[];
    isEditMode.value = false;
    ElMessage.success('价格已保存并重算');
  } catch (error) {
    ElMessage.error((error as Error).message || '保存失败');
  } finally {
    isLoading.value = false;
  }
};

const isExporting = ref(false);
const exportAll = async () => {
  isExporting.value = true;
  try {
    const resp = await wuyinApi.exportCheckout();
    if (resp?.url) window.open(`${BASE}${resp.url}`, '_blank');
  } catch (error) {
    ElMessage.error((error as Error).message || '导出失败');
  } finally {
    isExporting.value = false;
  }
};
const exportTag = async (tag: string) => {
  try {
    const resp = await wuyinApi.exportCheckout([tag]);
    if (resp?.url) window.open(`${BASE}${resp.url}`, '_blank');
  } catch (error) {
    ElMessage.error((error as Error).message || '导出失败');
  }
};

const fmt = (n: number) => n.toFixed(2);
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-4 flex-wrap">
      <div class="flex flex-col gap-2">
        <h2 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">吴饮结账 <span class="text-sm font-normal text-[var(--text-muted)]">按仓库标签分组；规格以斤记，结账按件计价（每条目 1 件）</span></h2>
      </div>
      <div class="flex items-center gap-3">
        <ElDatePicker v-model="dateRange" type="datetimerange" start-placeholder="开始时间" end-placeholder="结束时间" class="!w-80" :clearable="true" />
        <button @click="query" class="px-4 py-2.5 rounded-xl bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-sm flex items-center gap-2">
          <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': isLoading }" /> 查询
        </button>
        <template v-if="!isEditMode">
          <button @click="enterEditMode" :disabled="groups.length === 0" class="px-4 py-2.5 rounded-xl bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-secondary)] text-sm flex items-center gap-2 disabled:opacity-50">
            <Pencil class="w-4 h-4" /> 编辑单价
          </button>
          <button @click="exportAll" :disabled="groups.length === 0 || isExporting" class="px-6 py-2.5 rounded-xl bg-[var(--accent)] hover:bg-[var(--accent)] disabled:opacity-50 text-[var(--text-on-accent)] text-sm font-bold flex items-center gap-2 shadow-lg shadow-[var(--accent-glow)]">
            <Wallet class="w-4 h-4" /> 导出全部账单
          </button>
        </template>
        <template v-else>
          <button @click="isEditMode = false" class="px-4 py-2.5 rounded-xl bg-[var(--bg-hover)] text-[var(--text-muted)] text-sm flex items-center gap-2"><X class="w-4 h-4" /> 取消</button>
          <button @click="savePrices" :disabled="isLoading" class="px-6 py-2.5 rounded-xl bg-[var(--accent)] text-[var(--text-on-accent)] text-sm font-bold flex items-center gap-2 disabled:opacity-50">
            <Check class="w-4 h-4" /> {{ isLoading ? '保存中...' : '保存价格' }}
          </button>
        </template>
      </div>
    </div>

    <p v-if="groups.length === 0 && !isLoading" class="settings-card rounded-3xl px-6 py-12 text-center text-[var(--text-faint)]">选择时间范围后点击「查询」，每个仓库标签会生成一张独立账单</p>

    <section v-for="g in groups" :key="g.tag" class="settings-card rounded-3xl overflow-hidden">
      <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--border)]">
        <div class="flex items-center gap-3">
          <span class="bg-[var(--accent-soft-bg)] text-[var(--accent-soft)] text-sm font-bold px-3 py-1 rounded-full">{{ g.tag }}</span>
          <span class="text-xs text-[var(--text-muted)]">{{ g.count }} 单 · 合计 <span class="text-[var(--text-primary)] font-bold">￥{{ fmt(g.grandTotal) }}</span></span>
        </div>
        <button v-if="!isEditMode" @click="exportTag(g.tag)" class="px-4 py-1.5 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] text-xs flex items-center gap-1.5">
          <DollarSign class="w-3.5 h-3.5" /> 单独导出
        </button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-[var(--text-faint)] border-b border-[var(--border)]">
              <th class="px-4 py-3 font-semibold">商品</th>
              <th class="px-4 py-3 font-semibold">规格</th>
              <th v-for="d in g.dates" :key="d" class="px-4 py-3 font-semibold text-center">{{ d }}</th>
              <th class="px-4 py-3 font-semibold text-center">总件数</th>
              <th class="px-4 py-3 font-semibold text-center">单价(元/件)</th>
              <th class="px-4 py-3 font-semibold text-right">金额(元)</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!g.products || g.products.length === 0">
              <td :colspan="g.dates.length + 5" class="px-4 py-8 text-center text-[var(--text-faint)]">该标签下没有商品数据</td>
            </tr>
            <tr v-for="p in g.products" :key="p.name + p.spec" class="border-b border-[var(--border)] last:border-0">
              <td class="px-4 py-3 text-[var(--text-primary)] font-medium">{{ p.name }}</td>
              <td class="px-4 py-3 text-[var(--text-secondary)] font-mono text-xs">{{ p.spec }}</td>
              <td v-for="(n, i) in p.itemsPerDate" :key="i" class="px-4 py-3 text-center text-[var(--text-secondary)] font-mono" :class="{ 'text-[var(--text-faint)]': n === 0 }">{{ n === 0 ? '—' : n }}</td>
              <td class="px-4 py-3 text-center text-[var(--text-primary)] font-bold font-mono">{{ p.totalItems }}</td>
              <td class="px-4 py-3 text-center">
                <template v-if="isEditMode">
                  <input v-model.number="priceDraft[`${g.tag}|${p.name}|${p.spec}`]" type="number" step="0.01" min="0" class="glass-input px-2 py-1 rounded-lg w-20 text-center text-xs" />
                </template>
                <template v-else>
                  <span class="font-mono" :class="p.pricePerItem > 0 ? 'text-[var(--text-secondary)]' : 'text-[var(--text-faint)]'">{{ p.pricePerItem > 0 ? fmt(p.pricePerItem) : '未定价' }}</span>
                </template>
              </td>
              <td class="px-4 py-3 text-right font-mono" :class="p.totalAmount > 0 ? 'text-[var(--text-primary)]' : 'text-[var(--text-faint)]'">{{ fmt(p.totalAmount) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
