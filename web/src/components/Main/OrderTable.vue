<script setup lang="ts">
import { ref, reactive, computed } from 'vue';
import { Edit3, Trash2, Check, X, ScanSearch } from 'lucide-vue-next';
import { ElMessage } from 'element-plus';
import * as orderApi from '../../api/order';
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
      
      await orderApi.setPreview(excelAddressInfo);
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

// 分页相关状态
const currentPage = ref(1); // 当前页码
const pageSize = ref(10); // 每页显示条数
const pageSizeOptions = [5, 10, 20, 25]; // 每页显示条数选项

// 计算总页数
const totalPages = computed(() => {
  return Math.ceil(props.orders.length / pageSize.value);
});

// 计算当前页显示的订单
const paginatedOrders = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return props.orders.slice(start, end);
});

// 切换页码
const changePage = (page: number) => {
  if (page < 1 || page > totalPages.value) return;
  currentPage.value = page;
};

// 改变每页显示条数
const changePageSize = (size: number) => {
  pageSize.value = size;
  currentPage.value = 1; // 重置到第一页
};

// 上一页
const prevPage = () => {
  if (currentPage.value > 1) {
    changePage(currentPage.value - 1);
  }
};

// 下一页
const nextPage = () => {
  if (currentPage.value < totalPages.value) {
    changePage(currentPage.value + 1);
  }
};

// ===== 审核功能（T22，契约 §3.2）=====

// 审核状态：loading 标记与最近一次审核结果（审核为只读幂等端点，可重复点击刷新）
const auditLoading = ref(false);
const auditResult = ref<orderApi.AuditResponse | null>(null);

// diffType 展示元数据：missing 红 / wrong-key、wrong-spec 橙 / extra 黄 / normal 绿
const diffTypeMeta: Record<orderApi.AuditItem['diffType'], { label: string; badgeClass: string; itemClass: string }> = {
  missing: {
    label: '缺单',
    badgeClass: 'bg-red-500/10 text-red-400 border border-red-500/20',
    itemClass: 'border-l-2 border-l-red-500 bg-red-500/[0.02]',
  },
  'wrong-key': {
    label: '错单-关键字段',
    badgeClass: 'bg-orange-500/10 text-orange-400 border border-orange-500/20',
    itemClass: 'border-l-2 border-l-orange-500 bg-orange-500/[0.02]',
  },
  'wrong-spec': {
    label: '错单-规格',
    badgeClass: 'bg-orange-500/10 text-orange-400 border border-orange-500/20',
    itemClass: 'border-l-2 border-l-orange-500 bg-orange-500/[0.02]',
  },
  extra: {
    label: '多单',
    badgeClass: 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20',
    itemClass: 'border-l-2 border-l-yellow-500 bg-yellow-500/[0.02]',
  },
  normal: {
    label: '正常',
    badgeClass: 'bg-green-500/10 text-green-400 border border-green-500/20',
    itemClass: 'border-l-2 border-l-green-500 bg-green-500/[0.02]',
  },
};

// 触发审核：错误 ElMessage 展示，不静默
const runAudit = async () => {
  if (props.orders.length === 0 || auditLoading.value) return;
  auditLoading.value = true;
  try {
    auditResult.value = await orderApi.auditPreview();
  } catch (error) {
    console.error('审核失败:', error);
    ElMessage.error(error instanceof Error ? error.message : '审核请求失败');
  } finally {
    auditLoading.value = false;
  }
};

// 「真差异」项是否可应用修正：预览表中须存在对应行（缺单项 VID=0，无预览行可编辑）
const canApplyCorrection = (item: orderApi.AuditItem) => {
  return item.aiVerdict === '真差异' && item.VID > 0 && props.orders.some(o => o.id === String(item.VID));
};

// 应用修正：定位到对应 VID 的预览行、翻页使其可见并触发既有行内编辑。
// 修正落库仍走既有 setPreview，不新增端点（契约 §3.2 约定 4）。
const applyCorrection = (item: orderApi.AuditItem) => {
  const index = props.orders.findIndex(o => o.id === String(item.VID));
  if (index === -1) {
    ElMessage.warning('预览表中未找到对应行，请人工核对原始文本');
    return;
  }
  // 定位到目标行所在页并进入行内编辑态
  currentPage.value = Math.floor(index / pageSize.value) + 1;
  startEditing(props.orders[index]);
  if (item.aiSuggest) {
    ElMessage.info(`修正建议：${item.aiSuggest}`);
  }
};
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <h2 class="text-xl font-bold">数据预览</h2>
        <span class="px-2 py-0.5 rounded bg-teal-500/10 text-teal-500 text-[10px] font-bold uppercase tracking-wider">待导出</span>
      </div>

      <!-- 审核按钮（T22）：预览态为空时禁用；请求中 loading -->
      <!-- <button
        @click="runAudit"
        :disabled="orders.length === 0 || auditLoading"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors"
        :class="orders.length === 0 || auditLoading
          ? 'bg-white/5 text-slate-500 cursor-not-allowed'
          : 'bg-teal-500/10 text-teal-400 hover:bg-teal-500/20'"
        :title="orders.length === 0 ? '暂无预览数据，无法审核' : '对比原始文本与预览数据，AI 复核可疑差异项'"
      >
        <ScanSearch class="w-4 h-4" :class="{ 'animate-pulse': auditLoading }" />
        {{ auditLoading ? '审核中...' : '审核' }}
      </button> -->
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
              <th class="px-6 py-4">省份</th>
              <th class="px-6 py-4">规格</th>
              <th class="px-6 py-4">备注</th>
              <th class="px-6 py-4 w-32">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-white/5">
            <tr v-for="(order, index) in paginatedOrders" :key="order.id" :class="[index % 2 === 1 ? 'bg-white/[0.01]' : '', isEditing(order.id) ? 'bg-blue-500/10' : '']">
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

              <!-- 省份（后端从收货地址提取派生，行内编辑不提供修改） -->
              <td class="px-6 py-4">
                <span class="text-slate-300">{{ order.province || '-' }}</span>
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
            <tr v-if="paginatedOrders.length === 0">
              <td colspan="8" class="px-6 py-12 text-center text-slate-600">
                暂无识别数据，请在上方输入订单文本并点击识别
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <!-- Pagination -->
      <div class="p-6 border-t border-white/5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div class="flex flex-col sm:flex-row items-start sm:items-center gap-4 w-full sm:w-auto">
          <!-- 每页显示条数选择器 -->
          <div class="flex items-center gap-2">
            <label class="text-xs text-slate-500 font-medium">每页显示：</label>
            <select 
              v-model.number="pageSize" 
              @change="changePageSize(pageSize)"
              class="px-3 py-2 rounded-lg glass-input text-slate-300 text-xs transition-colors"
            >
              <option v-for="option in pageSizeOptions" :key="option" :value="option">{{ option }}</option>
            </select>
          </div>
          
          <!-- 显示信息 -->
          <p class="text-xs text-slate-500 font-medium">
            显示 {{ (currentPage - 1) * pageSize + 1 }} 到 {{ Math.min(currentPage * pageSize, orders.length) }} 条，共 {{ orders.length }} 条记录
          </p>
        </div>
        
        <!-- 页码导航 -->
        <div class="flex items-center gap-2 w-full sm:w-auto justify-center sm:justify-end">
          <!-- 上一页按钮 -->
          <button 
            @click="prevPage"
            :disabled="currentPage === 1"
            class="w-8 h-8 flex items-center justify-center rounded-lg bg-white/5 text-slate-500 hover:text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          
          <!-- 页码按钮 -->
          <template v-if="totalPages <= 5">
            <button 
              v-for="page in totalPages" 
              :key="page"
              @click="changePage(page)"
              class="w-8 h-8 flex items-center justify-center rounded-lg text-xs font-bold transition-colors"
              :class="currentPage === page ? 'bg-blue-600 text-white' : 'bg-white/5 text-slate-400 hover:text-white'"
            >
              {{ page }}
            </button>
          </template>
          <template v-else>
            <!-- 第一页 -->
            <button 
              @click="changePage(1)"
              class="w-8 h-8 flex items-center justify-center rounded-lg text-xs font-bold transition-colors"
              :class="currentPage === 1 ? 'bg-blue-600 text-white' : 'bg-white/5 text-slate-400 hover:text-white'"
            >
              1
            </button>
            
            <!-- 省略号 -->
            <span v-if="currentPage > 3" class="text-slate-600 text-xs mx-1 px-1">...</span>
            
            <!-- 当前页附近的页码 -->
            <!-- Vue3 中 v-if 优先级高于 v-for，须用 template 包裹使 v-if 进入 v-for 作用域 -->
            <template v-for="page in Math.max(2, currentPage - 1)" :key="page">
              <button
                v-if="page < currentPage && page > 1"
                @click="changePage(page)"
                class="w-8 h-8 flex items-center justify-center rounded-lg bg-white/5 text-slate-400 hover:text-white text-xs font-bold transition-colors"
              >
                {{ page }}
              </button>
            </template>
            
            <!-- 当前页 -->
            <button 
              class="w-8 h-8 flex items-center justify-center rounded-lg bg-blue-600 text-white text-xs font-bold"
            >
              {{ currentPage }}
            </button>
            
            <!-- 当前页附近的页码 -->
            <!-- Vue3 中 v-if 优先级高于 v-for，须用 template 包裹使 v-if 进入 v-for 作用域 -->
            <template v-for="page in Math.min(totalPages - 1, currentPage + 1)" :key="page">
              <button
                v-if="page > currentPage && page < totalPages"
                @click="changePage(page)"
                class="w-8 h-8 flex items-center justify-center rounded-lg bg-white/5 text-slate-400 hover:text-white text-xs font-bold transition-colors"
              >
                {{ page }}
              </button>
            </template>
            
            <!-- 省略号 -->
            <span v-if="currentPage < totalPages - 2" class="text-slate-600 text-xs mx-1 px-1">...</span>
            
            <!-- 最后一页 -->
            <button 
              @click="changePage(totalPages)"
              class="w-8 h-8 flex items-center justify-center rounded-lg text-xs font-bold transition-colors"
              :class="currentPage === totalPages ? 'bg-blue-600 text-white' : 'bg-white/5 text-slate-400 hover:text-white'"
            >
              {{ totalPages }}
            </button>
          </template>
          
          <!-- 下一页按钮 -->
          <button 
            @click="nextPage"
            :disabled="currentPage === totalPages"
            class="w-8 h-8 flex items-center justify-center rounded-lg bg-white/5 text-slate-500 hover:text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- 审核结果面板（T22，契约 §3.2） -->
    <div v-if="auditResult" class="rounded-2xl border border-white/5 bg-[#16191E] p-4 space-y-3">
      <div class="flex items-center justify-between gap-2 flex-wrap">
        <h3 class="text-base font-bold flex items-center gap-2">
          审核结果
          <!-- AI 复核环节异常时提示，程序对比结果仍有效 -->
          <span
            v-if="auditResult.aiFailed"
            class="px-2 py-0.5 rounded bg-yellow-500/10 text-yellow-400 text-[10px] font-bold"
          >
            AI 复核不可用，以下为程序对比结果
          </span>
        </h3>
        <span class="text-xs text-slate-500">AI 复核建议仅供参考，审核为只读操作，可重复点击「审核」刷新</span>
      </div>

      <!-- 汇总计数：缺单/错单/多单/正常 -->
      <div class="flex flex-wrap items-center gap-2 text-xs">
        <span class="px-2 py-1 rounded bg-red-500/10 text-red-400 font-medium">缺单 {{ auditResult.summary.missing }}</span>
        <span class="px-2 py-1 rounded bg-orange-500/10 text-orange-400 font-medium">错单 {{ auditResult.summary.wrong }}</span>
        <span class="px-2 py-1 rounded bg-yellow-500/10 text-yellow-400 font-medium">多单 {{ auditResult.summary.extra }}</span>
        <span class="px-2 py-1 rounded bg-green-500/10 text-green-400 font-medium">正常 {{ auditResult.summary.normal }}</span>
      </div>

      <!-- 差异明细列表：按 diffType 标色（missing 红 / wrong 橙 / extra 黄 / normal 绿） -->
      <ul v-if="auditResult.items && auditResult.items.length > 0" class="space-y-2">
        <li
          v-for="(item, idx) in auditResult.items"
          :key="idx"
          class="rounded-r-lg p-3 space-y-1.5"
          :class="diffTypeMeta[item.diffType].itemClass"
        >
          <div class="flex items-center justify-between gap-2 flex-wrap">
            <div class="flex items-center gap-2 flex-wrap text-sm">
              <span class="px-1.5 py-0.5 rounded text-[10px] font-bold" :class="diffTypeMeta[item.diffType].badgeClass">
                {{ diffTypeMeta[item.diffType].label }}
              </span>
              <span class="font-medium text-slate-200">{{ item['收件人'] || '-' }}</span>
              <span class="font-mono text-slate-400">{{ item['手机号'] || '-' }}</span>
              <span v-if="item.VID > 0" class="text-xs text-slate-500 font-mono">行 {{ item.VID }}</span>
            </div>
            <!-- 真差异项提供应用修正入口：定位对应 VID 行并进入既有行内编辑 -->
            <button
              v-if="canApplyCorrection(item)"
              @click="applyCorrection(item)"
              class="px-2.5 py-1 rounded-lg bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 transition-colors text-xs font-medium"
            >
              应用修正
            </button>
          </div>
          <p class="text-xs text-slate-400">{{ item['收货地址'] || '-' }}</p>
          <p class="text-xs text-slate-300">{{ item['商品规格'] || '-' }}</p>
          <p v-if="item.diffDesc" class="text-xs text-slate-400">{{ item.diffDesc }}</p>
          <p v-if="item.aiVerdict" class="text-xs">
            <span class="font-medium" :class="item.aiVerdict === '真差异' ? 'text-red-400' : 'text-green-400'">
              AI 判定：{{ item.aiVerdict }}
            </span>
            <span v-if="item.aiSuggest" class="text-slate-400">｜建议：{{ item.aiSuggest }}</span>
          </p>
        </li>
      </ul>
      <p v-else class="text-sm text-slate-500">暂无审核明细</p>
    </div>
  </section>
</template>

<style scoped>
.glass-input {
  background-color: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
}
.glass-input:focus {
  outline: none;
  border-color: #3B82F6;
  background-color: rgba(255, 255, 255, 0.06);
}
</style>
