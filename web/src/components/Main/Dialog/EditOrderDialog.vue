<template>
  <div v-if="show" class="fixed inset-0 bg-[var(--overlay)] flex items-center justify-center z-50 transition-opacity duration-300">
    <div class="bg-[var(--bg-sidebar)] border border-[var(--border-strong)] rounded-2xl p-8 max-w-lg w-full mx-4 flex flex-col max-h-[90vh] overflow-y-auto">
      <!-- Modal Header -->
      <div class="mb-6">
        <h3 class="text-xl font-bold text-[var(--text-primary)] font-display">修改订单</h3>
        <p class="text-[var(--text-muted)] text-sm mt-2">订单 ID：{{ order?.id ?? '-' }}，仅提交修改过的字段</p>
      </div>

      <!-- Modal Body -->
      <div class="flex-1 space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">收件人</label>
            <input type="text" v-model="form.name" placeholder="收件人姓名" class="dialog-input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">手机号</label>
            <input type="text" v-model="form.phone" placeholder="手机号" class="dialog-input" />
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">收货地址</label>
          <input type="text" v-model="form.address" placeholder="省、市、区、具体地址" class="dialog-input" />
          <p class="text-xs text-[var(--text-faint)] mt-1">地址变更后，系统将按新地址自动重提取省份</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">省份</label>
          <input type="text" v-model="form.province" placeholder="如：福建省" class="dialog-input" />
          <p class="text-xs text-[var(--text-faint)] mt-1">不修改本项时随收货地址自动识别；修改（含清空）后以提交值为准</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">商品规格</label>
          <input type="text" v-model="form.item" placeholder="如：【香水柠檬5斤】" class="dialog-input" />
          <p class="text-xs text-[var(--text-faint)] mt-1">不可清空，只能整体替换；多项用逗号分隔，如【香水柠檬5斤】,【葡萄柚10斤】</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-[var(--text-secondary)] mb-2">备注</label>
          <input type="text" v-model="form.remark" placeholder="备注（留空提交即清空）" class="dialog-input" />
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="mt-6 flex items-center justify-end gap-3">
        <button
          class="px-6 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] hover:text-[var(--text-primary)] font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="isSubmitting"
          @click="handleCancel"
        >
          取消
        </button>
        <button
          class="px-6 py-2 rounded-lg bg-[var(--accent)] hover:bg-[var(--accent)] text-[var(--text-on-accent)] font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="isSubmitting"
          @click="handleSave"
        >
          {{ isSubmitting ? '保存中...' : '保存修改' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 修改历史订单弹窗（T20）
// 保存流程：ElMessageBox.confirm 二次确认（文案含收件人+手机号）→ updateOrder(id, 只传修改字段) → 成功后通知父组件刷新列表。
// 接口契约：PUT /api/order/:id 中文键指针字段——未传=不更新、空串=清空（商品规格例外：不可清空只能整体替换），见契约 §3.3。
import { ref, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import * as orderApi from '../../../api/order';
import type { UpdateOrderRequest } from '../../../api/order';

// 可编辑订单数据（字段名与 OrderDetailsView 行对象一致，结构化兼容传入）
interface EditableOrder {
  id: number; // DB 主键，恒 > 0（id=0 预览行不可编辑，由父组件保证不打开）
  name: string; // 收件人
  phone: string; // 手机号
  address: string; // 收货地址
  province: string; // 省份全称（可为空串）
  item: string; // 商品规格展示串（只能整体替换输入）
  remark: string; // 备注（可为空串）
}

const props = defineProps<{
  show: boolean;
  order: EditableOrder | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'saved'): void; // 保存成功，父组件据此刷新列表
}>();

const isSubmitting = ref(false);

const form = ref({
  name: '',
  phone: '',
  address: '',
  province: '',
  item: '',
  remark: '',
});

// 弹窗打开时用行数据初始化表单
watch(
  () => [props.show, props.order] as const,
  ([show, order]) => {
    if (show && order !== null) {
      form.value = {
        name: order.name,
        phone: order.phone,
        address: order.address,
        province: order.province,
        item: order.item,
        remark: order.remark,
      };
      isSubmitting.value = false;
    }
  },
  { immediate: true },
);

const handleCancel = () => {
  if (!isSubmitting.value) {
    emit('close');
  }
};

const handleSave = async () => {
  const order = props.order;
  if (order === null || isSubmitting.value) return;

  const name = form.value.name.trim();
  const phone = form.value.phone.trim();
  const address = form.value.address.trim();
  const province = form.value.province.trim();
  const item = form.value.item.trim();
  const remark = form.value.remark.trim();

  // 商品规格不可清空，只能整体替换（契约 §3.3）
  if (item === '' && item !== order.item) {
    ElMessage.error('商品规格不能清空，只能整体替换');
    return;
  }

  // 仅收集修改过的字段（中文键指针语义：未传=不更新、空串=清空）
  const payload: UpdateOrderRequest = {};
  if (name !== order.name) payload['收件人'] = name;
  if (phone !== order.phone) payload['手机号'] = phone;
  if (address !== order.address) payload['收货地址'] = address;
  if (province !== order.province) payload['省份'] = province;
  if (item !== order.item) payload['商品规格'] = item;
  if (remark !== order.remark) payload['备注'] = remark;

  if (Object.keys(payload).length === 0) {
    ElMessage.info('未修改任何字段');
    return;
  }

  // 保存前二次确认（T4）：文案含收件人+手机号
  try {
    await ElMessageBox.confirm(
      `确认保存订单修改？收件人：${name}，手机号：${phone}`,
      '修改确认',
      {
        confirmButtonText: '确认修改',
        cancelButtonText: '取消',
        type: 'warning',
        customClass: 'order-action-msgbox',
      },
    );
  } catch {
    return; // 用户取消确认
  }

  isSubmitting.value = true;
  try {
    await orderApi.updateOrder(order.id, payload);
    ElMessage.success('订单修改成功');
    emit('saved');
    emit('close');
  } catch (error) {
    // 接口错误必须可见（T5）：request.ts 对 code!==0 抛出含后端 msg 的 Error，不得静默吞错
    ElMessage.error(`订单修改失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    isSubmitting.value = false;
  }
};
</script>

<style scoped>
/* 表单输入框统一暗色风格 */
.dialog-input {
  width: 100%;
  padding: 0.5rem 1rem;
  border-radius: 0.75rem;
  background-color: var(--bg-app);
  border: 1px solid var(--border-strong);
  color: var(--text-secondary);
  font-size: 0.875rem;
  outline: none;
  transition: all 0.2s ease;
}
.dialog-input::placeholder {
  color: var(--text-faint);
}
.dialog-input:focus {
  border-color: var(--accent);
}
.font-display {
  font-family: 'Cabinet Grotesk', sans-serif;
}
</style>
