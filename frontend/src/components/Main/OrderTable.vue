<script setup lang="ts">
import { ref, reactive, computed } from 'vue';
import { Edit3, Trash2, Check, X } from 'lucide-vue-next';
import { SetpreviewData } from '../../../wailsjs/go/main/App';
import type { Order } from '../../types';

const props = defineProps<{
  orders: Order[];
}>();

const emit = defineEmits<{
  (e: 'remove', id: string): void;
  (e: 'clearAll'): void;
  (e: 'edit', order: Order): void;
  (e: 'refreshData'): void;
  (e: 'delete', id: string): void;
}>();

// 编辑状态管理
const currentEditingId = ref<string | null>(null);
const editedOrder = reactive<Partial<Order>>({});
const validationErrors = reactive<Record<string, string>>({});

// 开始编辑
const startEditing = (order: Order) => {
  currentEditingId.value = order.id;
  // 复制订单数据到编辑对象
  Object.assign(editedOrder, order);
  // 重置验证错误
  Object.keys(validationErrors).forEach(key => delete validationErrors[key]);
};

// 保存编辑
const saveEdit = async () => {
  // 验证输入
  validateForm();
  
  if (Object.keys(validationErrors).length === 0 && currentEditingId.value) {
    try {
      // 转换为后端需要的 ExceladdressInfo 类型
      const excelAddressInfo = {
        VID: parseInt(editedOrder.id || '0'),
        "收件人": editedOrder.name || '',
        "手机号": editedOrder.phone || '',
        "收货地址": editedOrder.address || '',
        "商品规格": editedOrder.items || '',
        "备注": editedOrder.note || ''
      };
      
      await SetpreviewData(excelAddressInfo);
      emit('edit', editedOrder as Order);
      emit('refreshData');
      currentEditingId.value = null;
    } catch (error) {
      console.error('保存失败:', error);
      validationErrors.general = '保存失败，请重试';
    }
  }
};

// 取消编辑
const cancelEdit = () => {
  currentEditingId.value = null;
  Object.keys(editedOrder).forEach(key => delete editedOrder[key]);
  Object.keys(validationErrors).forEach(key => delete validationErrors[key]);
};

// 验证表单
const validateForm = () => {
  Object.keys(validationErrors).forEach(key => delete validationErrors[key]);
  
  if (!editedOrder.name?.trim()) {
    validationErrors.name = '客户姓名不能为空';
  }
  
  if (!editedOrder.phone?.trim()) {
    validationErrors.phone = '联系电话不能为空';
  } else if (!/^1[3-9]\d{9}$/.test(editedOrder.phone)) {
    validationErrors.phone = '请输入有效的手机号码';
  }
  
  if (!editedOrder.address?.trim()) {
    validationErrors.address = '收货地址不能为空';
  }
  
  if (!editedOrder.items?.trim()) {
    validationErrors.items = '商品项不能为空';
  }
};

// 计算属性：检查是否正在编辑
const isEditing = (orderId: string) => {
  return currentEditingId.value === orderId;
};

// 计算属性：过滤掉general错误的验证错误
const filteredValidationErrors = computed(() => {
  const errors = { ...validationErrors };
  delete errors.general;
  return errors;
});
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <h2 class="text-xl font-bold">数据预览</h2>
        <span class="px-2 py-0.5 rounded bg-teal-500/10 text-teal-500 text-[10px] font-bold uppercase tracking-wider">待导出</span>
      </div>

    </div>
    
    <!-- 验证错误提示 -->
    <div v-if="Object.keys(validationErrors).length > 0" class="rounded-lg bg-red-500/10 border border-red-500/20 p-4">
      <div class="flex items-start gap-3">
        <div class="text-red-400">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-.77-1.964-.77-2.732 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <div class="flex-1">
          <h3 class="text-red-400 font-medium mb-1">验证错误</h3>
          <ul class="text-sm text-red-300 space-y-1">
            <li v-for="[key, error] in Object.entries(filteredValidationErrors)" :key="key">
              {{ error }}
            </li>
            <li v-if="validationErrors.general">
              {{ validationErrors.general }}
            </li>
          </ul>
        </div>
      </div>
    </div>
    
    <div class="rounded-2xl border border-white/5 bg-[#16191E] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm border-collapse">
          <thead class="bg-white/5 text-slate-400 font-medium">
            <tr>
              <th class="px-6 py-4">编号</th>
              <th class="px-6 py-4">收件人</th>
              <th class="px-6 py-4">电话</th>
              <th class="px-6 py-4">地址</th>
              <th class="px-6 py-4">规格</th>
              <th class="px-6 py-4">备注</th>
              <th class="px-6 py-4 w-32">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-white/5">
            <tr v-for="(order, index) in orders" :key="order.id" :class="[index % 2 === 1 ? 'bg-white/[0.01]' : '', isEditing(order.id) ? 'bg-blue-500/10' : '']">
              <!-- 编号 -->
              <td class="px-6 py-4 font-mono text-slate-500">{{ order.id }}</td>
              
              <!-- 客户姓名 -->
              <td class="px-6 py-4">
                <template v-if="isEditing(order.id)">
                  <input 
                    v-model="editedOrder.name" 
                    type="text" 
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors"
                    placeholder="收件人"
                  />
                  <span v-if="validationErrors.name" class="text-red-400 text-xs mt-1 block">{{ validationErrors.name }}</span>
                </template>
                <template v-else>
                  {{ order.name }}
                </template>
              </td>
              
              <!-- 联系电话 -->
              <td class="px-6 py-4">
                <template v-if="isEditing(order.id)">
                  <input 
                    v-model="editedOrder.phone" 
                    type="text" 
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors font-mono"
                    placeholder="电话"
                  />
                  <span v-if="validationErrors.phone" class="text-red-400 text-xs mt-1 block">{{ validationErrors.phone }}</span>
                </template>
                <template v-else>
                  <span class="font-mono text-slate-300">{{ order.phone }}</span>
                </template>
              </td>
              
              <!-- 收货地址 -->
              <td class="px-6 py-4">
                <template v-if="isEditing(order.id)">
                  <input 
                    v-model="editedOrder.address" 
                    type="text" 
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors"
                    placeholder="地址"
                  />
                  <span v-if="validationErrors.address" class="text-red-400 text-xs mt-1 block">{{ validationErrors.address }}</span>
                </template>
                <template v-else>
                  <span class="text-slate-300 max-w-xs truncate">{{ order.address }}</span>
                </template>
              </td>
              
              <!-- 商品项 -->
              <td class="px-6 py-4">
                <template v-if="isEditing(order.id)">
                  <input 
                    v-model="editedOrder.items" 
                    type="text" 
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors"
                    placeholder="规格"
                  />
                  <span v-if="validationErrors.items" class="text-red-400 text-xs mt-1 block">{{ validationErrors.items }}</span>
                </template>
                <template v-else>
                  {{ order.items }}
                </template>
              </td>
              
              <!-- 备注 -->
              <td class="px-6 py-4">
                <template v-if="isEditing(order.id)">
                  <input 
                    v-model="editedOrder.note" 
                    type="text" 
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors"
                    placeholder="备注"
                  />
                </template>
                <template v-else>
                  <span class="text-slate-500">{{ order.note || '-' }}</span>
                </template>
              </td>
              
              <!-- 操作 -->
              <td class="px-6 py-4">
                <template v-if="isEditing(order.id)">
                  <div class="flex gap-2">
                    <button 
                      @click="saveEdit"
                      class="p-1.5 rounded bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors"
                      title="保存"
                    >
                      <Check class="w-4 h-4" />
                    </button>
                    <button 
                      @click="cancelEdit"
                      class="p-1.5 rounded bg-white/5 text-slate-400 hover:text-white transition-colors"
                      title="取消"
                    >
                      <X class="w-4 h-4" />
                    </button>
                  </div>
                </template>
                <template v-else>
                  <div class="flex gap-2">
                    <button 
                      @click="startEditing(order)"
                      class="p-1.5 rounded bg-white/5 text-slate-400 hover:text-blue-400 transition-colors"
                      title="编辑"
                    >
                      <Edit3 class="w-4 h-4" />
                    </button>
                    <button 
                      @click="$emit('delete', order.id)"
                      class="p-1.5 rounded bg-white/5 text-slate-400 hover:text-red-400 transition-colors"
                      title="删除"
                    >
                      <Trash2 class="w-4 h-4" />
                    </button>
                  </div>
                </template>
              </td>
            </tr>
            <tr v-if="orders.length === 0">
              <td colspan="7" class="px-6 py-12 text-center text-slate-600">
                暂无识别数据，请在上方输入订单文本并点击识别
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
