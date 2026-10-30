<script setup lang="ts">
import { ref, computed } from 'vue';
import { ElMessage } from 'element-plus';
import * as wuyinApi from '../../api/wuyin';
import type { WuyinOrder } from '../../types';

// 吴饮预览表格：标签列内联下拉（改动即保存），其余字段行内编辑（与一点点
// OrderTable 的行编辑模式同构），删除走 VID。

const props = defineProps<{
  orders: WuyinOrder[];
  tags: string[]; // 可选标签列表（含「未分配」兜底）
}>();

const emit = defineEmits<{
  (e: 'delete', vid: number): void;
  (e: 'refresh'): void;
}>();

const currentEditingId = ref<string | null>(null);
const editedOrder = ref<Partial<WuyinOrder>>({});
const validationErrors = ref<Record<string, string>>({});

const startEditing = (order: WuyinOrder) => {
  currentEditingId.value = order.id;
  editedOrder.value = { ...order };
  validationErrors.value = {};
};

const cancelEdit = () => {
  currentEditingId.value = null;
  editedOrder.value = {};
  validationErrors.value = {};
};

const validate = () => {
	const errs: Record<string, string> = {};
	if (!editedOrder.value.name?.trim()) errs.name = '收件人不能为空';
	if (!editedOrder.value.address?.trim()) errs.address = '收货地址不能为空';
	if (!editedOrder.value.items?.trim()) errs.items = '商品规格不能为空';
	// 商品规格只认斤（"件"只体现在结账计件），每个【】条目即 1 件
	else if (!/【[^】]+\d+(?:斤)?】/.test(editedOrder.value.items) || /【[^】]*\d+件】/.test(editedOrder.value.items)) errs.items = '格式如【香水柠檬9斤】，两件写两个条目';
	validationErrors.value = errs;
	return Object.keys(errs).length === 0;
};

const saveEdit = async () => {
  if (!validate() || currentEditingId.value === null) return;
  try {
    await wuyinApi.setPreview({
      VID: parseInt(currentEditingId.value),
      收件人: editedOrder.value.name || '',
      手机号: editedOrder.value.phone || '',
      收货地址: editedOrder.value.address || '',
      商品规格: editedOrder.value.items || '',
      备注: editedOrder.value.note || '',
      标签: editedOrder.value.tag || '',
      发送人: editedOrder.value.sender || '',
    });
    ElMessage.success('保存成功');
    cancelEdit();
    emit('refresh');
  } catch (error) {
    ElMessage.error((error as Error).message || '保存失败，请重试');
  }
};

// 标签切换即保存（不改其他字段：后端 SetPreview 全量覆盖，故带上当前行原值）
const changeTag = async (order: WuyinOrder, tag: string) => {
  try {
    await wuyinApi.setPreview({
      VID: parseInt(order.id),
      收件人: order.name,
      手机号: order.phone,
      收货地址: order.address,
      商品规格: order.items,
      备注: order.note,
      标签: tag,
      发送人: order.sender,
    });
    order.tag = tag;
  } catch (error) {
    ElMessage.error((error as Error).message || '标签保存失败');
  }
};

// 分页
const currentPage = ref(1);
const pageSize = ref(10);
const paginatedOrders = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  return props.orders.slice(start, start + pageSize.value);
});
const totalPages = computed(() => Math.ceil(props.orders.length / pageSize.value) || 1);
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-bold tracking-tight text-[var(--text-primary)]">识别预览 <span class="text-sm font-normal text-[var(--text-muted)]">共 {{ orders.length }} 条</span></h2>
    </div>

    <div class="settings-card rounded-3xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-[var(--text-faint)] border-b border-[var(--border)]">
              <th class="px-4 py-3 font-semibold">#</th>
              <th class="px-4 py-3 font-semibold">标签</th>
              <th class="px-4 py-3 font-semibold">收件人</th>
              <th class="px-4 py-3 font-semibold">手机号</th>
              <th class="px-4 py-3 font-semibold">收货地址</th>
              <th class="px-4 py-3 font-semibold">商品规格</th>
              <th class="px-4 py-3 font-semibold">发送人</th>
              <th class="px-4 py-3 font-semibold">备注</th>
              <th class="px-4 py-3 font-semibold text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="paginatedOrders.length === 0">
              <td colspan="9" class="px-4 py-12 text-center text-[var(--text-faint)]">暂无数据，请先识别或导入</td>
            </tr>
            <tr v-for="order in paginatedOrders" :key="order.id" class="border-b border-[var(--border)] last:border-0 hover:bg-[var(--bg-hover)] transition-colors">
              <!-- 编辑态 -->
              <template v-if="currentEditingId === order.id">
                <td class="px-4 py-3 text-[var(--text-muted)]">{{ order.id }}</td>
                <td class="px-4 py-3">
                  <select v-model="editedOrder.tag" class="glass-input px-2 py-1.5 rounded-lg text-xs">
                    <option v-for="t in tags" :key="t" :value="t">{{ t }}</option>
                  </select>
                </td>
                <td class="px-4 py-3"><input v-model="editedOrder.name" class="glass-input px-2 py-1.5 rounded-lg w-24 text-xs" /></td>
                <td class="px-4 py-3"><input v-model="editedOrder.phone" class="glass-input px-2 py-1.5 rounded-lg w-32 text-xs" /></td>
                <td class="px-4 py-3"><textarea v-model="editedOrder.address" rows="2" class="glass-input px-2 py-1.5 rounded-lg w-64 text-xs"></textarea></td>
                <td class="px-4 py-3">
                  <input v-model="editedOrder.items" class="glass-input px-2 py-1.5 rounded-lg w-36 text-xs font-mono" />
                  <p v-if="validationErrors.items" class="text-[10px] text-red-400 mt-1">{{ validationErrors.items }}</p>
                </td>
                <td class="px-4 py-3"><input v-model="editedOrder.sender" class="glass-input px-2 py-1.5 rounded-lg w-28 text-xs" /></td>
                <td class="px-4 py-3"><input v-model="editedOrder.note" class="glass-input px-2 py-1.5 rounded-lg w-24 text-xs" /></td>
                <td class="px-4 py-3 text-right space-x-2 whitespace-nowrap">
                  <button @click="saveEdit" class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--text-on-accent)] text-xs font-bold">保存</button>
                  <button @click="cancelEdit" class="px-3 py-1.5 rounded-lg bg-[var(--bg-hover)] text-[var(--text-muted)] text-xs">取消</button>
                </td>
              </template>
              <!-- 展示态 -->
              <template v-else>
                <td class="px-4 py-3 text-[var(--text-faint)]">{{ order.id }}</td>
                <td class="px-4 py-3">
                  <select
                    :value="order.tag"
                    @change="changeTag(order, ($event.target as HTMLSelectElement).value)"
                    class="bg-[var(--accent-soft-bg)] text-[var(--accent-soft)] text-xs font-medium px-2.5 py-1 rounded-full border-0 cursor-pointer outline-none"
                  >
                    <option v-for="t in tags" :key="t" :value="t">{{ t }}</option>
                    <option v-if="!tags.includes(order.tag)" :value="order.tag">{{ order.tag }}</option>
                  </select>
                </td>
                <td class="px-4 py-3 text-[var(--text-primary)] font-medium">{{ order.name || '—' }}</td>
                <td class="px-4 py-3 text-[var(--text-secondary)] font-mono text-xs">{{ order.phone || '—' }}</td>
                <td class="px-4 py-3 text-[var(--text-secondary)] max-w-xs truncate" :title="order.address">{{ order.address }}</td>
                <td class="px-4 py-3 text-[var(--text-secondary)] font-mono text-xs whitespace-nowrap">{{ order.items }}</td>
                <td class="px-4 py-3 text-[var(--text-muted)] text-xs max-w-[10rem] truncate" :title="order.sender">{{ order.sender || '—' }}</td>
                <td class="px-4 py-3 text-[var(--text-muted)] text-xs max-w-[8rem] truncate" :title="order.note">{{ order.note || '—' }}</td>
                <td class="px-4 py-3 text-right space-x-2 whitespace-nowrap">
                  <button @click="startEditing(order)" class="px-3 py-1.5 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] text-xs">编辑</button>
                  <button @click="$emit('delete', parseInt(order.id))" class="px-3 py-1.5 rounded-lg bg-[var(--bg-hover)] hover:bg-red-500/10 hover:text-red-400 text-[var(--text-muted)] text-xs">删除</button>
                </td>
              </template>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div v-if="orders.length > pageSize" class="flex items-center justify-between px-4 py-3 border-t border-[var(--border)] text-xs text-[var(--text-muted)]">
        <span>第 {{ currentPage }} / {{ totalPages }} 页</span>
        <div class="space-x-2">
          <button @click="currentPage = Math.max(1, currentPage - 1)" :disabled="currentPage <= 1" class="px-3 py-1 rounded-lg bg-[var(--bg-hover)] disabled:opacity-40">上一页</button>
          <button @click="currentPage = Math.min(totalPages, currentPage + 1)" :disabled="currentPage >= totalPages" class="px-3 py-1 rounded-lg bg-[var(--bg-hover)] disabled:opacity-40">下一页</button>
        </div>
      </div>
    </div>
  </section>
</template>
