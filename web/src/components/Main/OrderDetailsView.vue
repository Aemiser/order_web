<script setup lang="ts">
import { ref, computed, watch, onMounted, defineAsyncComponent } from 'vue';
import {
  RotateCw,
  Download,
  ChevronDown,
  ChevronRight,
  User,
  Phone,
  MapPin,
  Package,
  Calendar,
  RotateCcw,
  Search,
  Edit2,
  Trash2,
  ChevronLeft
} from 'lucide-vue-next';
import * as orderApi from '../../api/order';
import type { ExcelAddressInfo } from '../../api/order';
import { ElDatePicker, ElMessage, ElMessageBox } from 'element-plus';
import * as XLSX from "xlsx";
import EditOrderDialog from './Dialog/EditOrderDialog.vue';

// 省份订单分布地图（T19）：echarts 体积大，defineAsyncComponent 懒加载分包
const ProvinceMap = defineAsyncComponent(() => import('./Charts/ProvinceMap.vue'));

const props = defineProps<{
  activeTab?: string;
}>();

// 监听 activeTab 变化，当切换到详情页时调用 resetQuery
watch(() => props.activeTab, (newTab) => {
  if (newTab === 'details') {
    resetQuery();
  }
});

// 组件挂载时调用 resetQuery
onMounted(() => {
  resetQuery();
});

const isExpanded = ref(true);

const toggleQuery = () => {
  isExpanded.value = !isExpanded.value;
};

// 分页相关状态
const pageSize = ref(10); // 每页显示条数
const currentPage = ref(1); // 当前页码
const totalPages = ref(1); // 总页数
const isLoading = ref(false); // 加载状态

// 每页显示条数选项
const pageSizeOptions = [5, 10, 20,50,100];

// 查询参数
const searchParams = ref({
  name: '',
  phone: '',
  address: '',
  item: '',
  keyword: '',
  startTime: '',
  endTime: '',
});

// 日期范围
const dateRange = ref<[string, string] | null>(null);

// 错误信息
const errors = ref({
  dateRange: ''
});

// 处理日期变化
const handleDateChange = () => {
  // 清除错误信息
  errors.value.dateRange = '';
  
  // 更新 searchParams 中的 startTime 和 endTime
  if (dateRange.value && dateRange.value.length === 2) {
    searchParams.value.startTime = dateRange.value[0];
    searchParams.value.endTime = dateRange.value[1];
  } else {
    searchParams.value.startTime = '';
    searchParams.value.endTime = '';
  }
};

// 验证日期范围
const validateDateRange = () => {
  // 检查日期范围是否有效
  if (dateRange.value && dateRange.value.length === 2) {
    const startDate = new Date(dateRange.value[0]);
    const endDate = new Date(dateRange.value[1]);
    
    if (endDate < startDate) {
      errors.value.dateRange = '截至时间不能早于起始时间';
      return false;
    }
  }
  errors.value.dateRange = '';
  return true;
};

// 移除日期变化的自动监听，只在点击查询按钮时进行验证

// ---- T19 省份分布地图联动时间参数 ----
// ProvinceMap 的 props 为可选 RFC3339 字符串：都不传=全量统计，只传其一=后端 40000（契约 §3.1）。
// 故与订单查询同口径归一化为「起始日 00:00:00 ~ 截止日 23:59:59」成对给出，或全不传。
const provinceMapTimeRange = computed<{ startTime?: string; endTime?: string }>(() => {
  if (!searchParams.value.startTime || !searchParams.value.endTime) {
    return {};
  }
  const startDate = new Date(searchParams.value.startTime);
  startDate.setHours(0, 0, 0, 0);
  const endDate = new Date(searchParams.value.endTime);
  endDate.setHours(23, 59, 59, 999);
  return { startTime: startDate.toISOString(), endTime: endDate.toISOString() };
});

// 订单详情行（详情页局部类型，勿引 any）：
// id：DB 主键（预览行/占位行为 0，id>0 才可编辑/删除，契约 §3.3）；
// province：省份全称（查询出参「省份」/Province，未识别为空串）；
// name/phone/address/item/remark 存原始值，展示占位在模板内回退（编辑弹窗需读原始值）。
interface OrderDetailRow {
  id: number;
  name: string;
  phone: string;
  address: string;
  province: string;
  item: string;
  quantity: number;
  time: string;
  color: string;
  remark: string;
}

// 订单明细行（真实数据由挂载后的 resetQuery → fetchOrderData 拉取填充，初始为空）
const orderDetails = ref<OrderDetailRow[]>([]);

// 总订单数
const totalOrders = ref(0);

// 防抖函数
const debounce = (func: Function, delay: number) => {
  let timeoutId: ReturnType<typeof setTimeout>;
  return (...args: any[]) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => func.apply(null, args), delay);
  };
};

// 关键词搜索防抖
const handleKeywordSearch = debounce(() => {
  fetchOrderData();
}, 500);

// 计算总页数
const calculateTotalPages = () => {
  totalPages.value = Math.ceil(totalOrders.value / pageSize.value);
  // 确保当前页码不超过总页数
  if (currentPage.value > totalPages.value) {
    currentPage.value = Math.max(1, totalPages.value);
  }
};

// 计算展示的页码列表
type PageItem = { type: 'page' | 'ellipsis'; value: number };

const displayPages = computed(() => {
  const total = totalPages.value;
  const current = currentPage.value;
  const pages: PageItem[] = [];

  if (total <= 5) {
    for (let i = 1; i <= total; i++) {
      pages.push({ type: 'page', value: i });
    }
    return pages;
  }

  pages.push({ type: 'page', value: 1 });

  const start = Math.max(2, current - 3);
  const end = Math.min(total - 1, current + 3);

  if (start > 2) {
    pages.push({ type: 'ellipsis', value: -1 });
  }

  for (let i = start; i <= end; i++) {
    pages.push({ type: 'page', value: i });
  }

  if (end < total - 1) {
    pages.push({ type: 'ellipsis', value: -1 });
  }

  pages.push({ type: 'page', value: total });

  return pages;
});

// 切换页码
const changePage = debounce((page: number) => {
  if (page < 1 || page > totalPages.value) return;
  currentPage.value = page;
  fetchOrderData();
}, 300);

// 改变每页显示条数
const changePageSize = debounce((size: number) => {
  pageSize.value = size;
  currentPage.value = 1; // 重置到第一页
  calculateTotalPages();
  fetchOrderData();
}, 300);

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

// 获取订单数据
const fetchOrderData = async () => {
  isLoading.value = true;
  try {
    // 验证日期范围
    if (!validateDateRange()) {
      return;
    }
    
    // 构建 addresssinfo 结构体
    const addresssinfo = {
      "收件人": searchParams.value.name || '',
      "手机号": searchParams.value.phone || '',
      "收货地址": searchParams.value.address || '',
      "备注": ''
    };
    
    // 创建完整的请求参数，包含 addresssinfo 字段
    const requestParams: Record<string, unknown> = {
      limit: pageSize.value,
      page: currentPage.value,
      key: searchParams.value.keyword || '',
      order: '',
      addresssinfo: addresssinfo
    };
    
    // 只有当日期不为空时才添加到请求参数中
    if (searchParams.value.startTime) {
      // 将日期设置为当天的 00:00:00
      const startDate = new Date(searchParams.value.startTime);
      startDate.setHours(0, 0, 0, 0);
      requestParams.Starttime = startDate.toISOString();
    }
    if (searchParams.value.endTime) {
      // 将日期设置为当天的 23:59:59
      const endDate = new Date(searchParams.value.endTime);
      endDate.setHours(23, 59, 59, 999);
      requestParams.Endtime = endDate.toISOString();
    }
    
    // 调用后端订单查询接口
    const result = await orderApi.queryOrders(requestParams);

    // 检查返回值类型
    if (result && typeof result === 'object') {
      // 检查是否包含 list 和 count 字段
      if (Array.isArray(result.list) && (typeof result.count === 'number' || typeof result.count === 'string')) {

        // 更新总订单数
        totalOrders.value = Number(result.count);
        calculateTotalPages();

        // 根据返回的结果更新订单列表（原始值入行，展示占位由模板回退；F4：中文键+英文别名双写读取）
        orderDetails.value = result.list.map((item: ExcelAddressInfo) => ({
          id: item.id ?? item.ID ?? 0, // DB 主键：预览行 0、历史订单 >0（契约 §3.3）
          name: item['收件人'] ?? item.Recipient ?? '',
          phone: item['手机号'] ?? item.Phone ?? '',
          address: item['收货地址'] ?? item.Address ?? '',
          province: item['省份'] ?? item.Province ?? '',
          item: item['商品规格'] ?? item.ProductSpecs ?? '',
          quantity: 1, // 后端返回的数据中可能没有数量字段，默认设为 1
          time: item.createdAt ? new Date(item.createdAt).toLocaleString() : '未知时间',
          color: 'blue', // 默认颜色为蓝色
          remark: item['备注'] ?? item.Remarks ?? ''
        }));
      } else if (result.list == null){
        // 说明没找到数据
        orderDetails.value = [];
        totalOrders.value = Number(result.count);
        calculateTotalPages();
      }else{
        // 如果格式不正确，将订单列表设为空，总订单数保持不变
        orderDetails.value = [];
        // 不更新总订单数，保持之前的值
        calculateTotalPages();
      }
    } else {
      // 如果返回的是其他类型，将订单列表设为空，总订单数保持不变
      orderDetails.value = [];
      // 不更新总订单数，保持之前的值
      calculateTotalPages();
    }
  } catch (error) {
    console.error('查询订单失败:', error);
    // 发生错误时，将订单列表设为空，总订单数保持不变
    orderDetails.value = [];
    // 不更新总订单数，保持之前的值
    calculateTotalPages();
  } finally {
    isLoading.value = false;
  }
};

const exportToExcel = async () => {
  // 导出前校验日期范围：未选择或不可解析时中断，
  // 避免 new Date('') 得 Invalid Date 后 toISOString() 抛 RangeError 被 catch 静默吞掉
  if (!searchParams.value.startTime || !searchParams.value.endTime) {
    ElMessage.warning('请先选择日期范围后再导出');
    return;
  }
  const startDate = new Date(searchParams.value.startTime);
  const endDate = new Date(searchParams.value.endTime);
  if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
    ElMessage.warning('日期范围无效，请重新选择后再导出');
    return;
  }
  if (endDate < startDate) {
    ElMessage.warning('截至时间不能早于起始时间');
    return;
  }

  isLoading.value = true;
  try {
    const addresssinfo = {
      "收件人": searchParams.value.name || '',
      "手机号": searchParams.value.phone || '',
      "收货地址": searchParams.value.address || '',
      "备注": ''
    };

    const requestParams: Record<string, unknown> = {
      limit: 0,   // limit=0 获取全部数据
      page: 1,
      key: searchParams.value.keyword || '',
      order: '',
      addresssinfo: addresssinfo
    };

    startDate.setHours(0, 0, 0, 0);
    requestParams.Starttime = startDate.toISOString();
    endDate.setHours(23, 59, 59, 999);
    requestParams.Endtime = endDate.toISOString();

    const result = await orderApi.queryOrders(requestParams);

    if (result && Array.isArray(result.list)) {
      const exportData = result.list.map((item: ExcelAddressInfo) => ({
        '日期': item.createdAt ? new Date(item.createdAt).toLocaleString() : '未知时间',
        '收件人': item['收件人'] || '',
        '手机号': item['手机号'] || '',
        '收货地址': item['收货地址'] || '',
        '商品规格': item['商品规格'] || '',
        '备注': item['备注'] || ''
      }));

      const worksheet = XLSX.utils.json_to_sheet(exportData);
      const workbook = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(workbook, worksheet, "Orders");
      // XLSX.writeFile(workbook, `Orders_${new Date().toISOString().split('T')[0]}详情导出.xlsx`);
      startDate.setDate(startDate.getDate() + 1);
      XLSX.writeFile(workbook, `Orders_${startDate.toISOString().split('T')[0]}-${endDate.toISOString().split('T')[0]}.xlsx`);
    }
  } catch (error) {
    console.error('导出订单失败:', error);
    ElMessage.error('导出订单失败，请稍后重试');
  } finally {
    isLoading.value = false;
  }
};


// ---- T20 编辑：弹窗承载表单（EditOrderDialog），保存成功后刷新列表 ----
const isEditDialogVisible = ref(false);
const editingOrder = ref<OrderDetailRow | null>(null);

const openEditDialog = (order: OrderDetailRow) => {
  if (order.id <= 0) return; // 契约 §3.3：仅 id>0 的历史订单可删改，预览行（id=0）不可
  editingOrder.value = order;
  isEditDialogVisible.value = true;
};

const closeEditDialog = () => {
  isEditDialogVisible.value = false;
};

const handleEditSaved = () => {
  // 修改保存成功 → 刷新列表（收货地址变更后省份由后端重提取，刷新即可见）
  fetchOrderData();
};

// ---- T20 删除：ElMessageBox.confirm 二次确认（文案含收件人+手机号+地址）→ deleteOrder → 移除该行 ----
const deletingId = ref<number | null>(null);

const handleDeleteOrder = async (order: OrderDetailRow) => {
  if (order.id <= 0 || deletingId.value !== null) return;

  try {
    await ElMessageBox.confirm(
      `确认删除该订单？删除后不可恢复。收件人：${order.name}，手机号：${order.phone}，收货地址：${order.address}`,
      '删除确认',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
        customClass: 'order-action-msgbox',
      },
    );
  } catch {
    return; // 用户取消确认
  }

  deletingId.value = order.id;
  try {
    await orderApi.deleteOrder(order.id);
    // 删除端点非幂等（重复删除返回 50000），成功后立即从列表移除该行避免重放
    orderDetails.value = orderDetails.value.filter((row) => row.id !== order.id);
    totalOrders.value = Math.max(0, totalOrders.value - 1);
    calculateTotalPages();
    ElMessage.success('订单已删除');
  } catch (error) {
    // 接口错误必须可见（T5）：如重复删除时后端 50000「订单不存在」
    ElMessage.error(`删除订单失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    deletingId.value = null;
  }
};

const getTagClass = (color: string) => {
  switch (color) {
    case 'blue': return 'bg-[var(--accent-soft-bg)] text-[var(--accent-soft)]';
    case 'teal': return 'bg-[var(--teal-soft)] text-[var(--teal)]';
    case 'purple': return 'bg-purple-500/10 text-purple-400';
    default: return 'bg-slate-500/10 text-[var(--text-muted)]';
  }
};

const resetQuery = async () => {
  // 重置分页参数
  currentPage.value = 1;
  pageSize.value = 10;
  
  // 重置搜索参数
  searchParams.value = {
    name: '',
    phone: '',
    address: '',
    item: '',
    keyword: '',
    startTime: '',
    endTime: ''
  };
  
  // 调用 fetchOrderData 获取数据
  await fetchOrderData();
};
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <h2 class="text-3xl font-bold font-display text-[var(--text-primary)]">订单详情管理</h2>

    </div>

    <!-- Advanced Query Section -->
    <section class="bg-[var(--bg-sidebar)] border border-[var(--border)] rounded-2xl p-4 shadow-sm">
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="w-1.5 h-6 bg-[var(--accent)] rounded-full"></div>
          <h3 class="text-lg font-bold font-display">高级查询筛选</h3>
        </div>
        <div class="flex items-center gap-3">
          <span class="text-xs text-[var(--text-faint)] font-medium tracking-wide">共计 {{ totalOrders }} 条订单记录</span>
          <button 
            @click="toggleQuery"
            class="flex items-center gap-2 p-2 px-3 rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-all text-xs font-medium border border-[var(--border)] active:scale-95"
          >
            <span>{{ isExpanded ? '收起筛选' : '展开筛选' }}</span>
            <ChevronDown 
              class="w-4 h-4 transition-transform duration-300" 
              :class="{ 'rotate-180': isExpanded }"
            />
          </button>
        </div>
      </div>
      
      <div 
        class="transition-all duration-300 ease-in-out overflow-hidden" 
        :style="{ maxHeight: isExpanded ? '400px' : '0px', opacity: isExpanded ? '1' : '0', marginTop: isExpanded ? '0px' : '-15px' }"
      >
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-4">

          <div class="space-y-2">
            <label class="text-[10px] font-bold text-[var(--text-faint)] uppercase tracking-[0.1em] flex items-center gap-2">
              客户姓名
            </label>
            <div class="relative">
              <User class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--accent)]/60" />
              <input type="text" v-model="searchParams.name" placeholder="搜索姓名..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-[var(--text-secondary)]">
            </div>
          </div>
          
          <div class="space-y-2">
            <label class="text-[10px] font-bold text-[var(--text-faint)] uppercase tracking-[0.1em] flex items-center gap-2">
              联系电话
            </label>
            <div class="relative">
              <Phone class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--accent)]/60" />
              <input type="text" v-model="searchParams.phone" placeholder="手机号或固话..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-[var(--text-secondary)]">
            </div>
          </div>
          
          <div class="space-y-2">
            <label class="text-[10px] font-bold text-[var(--text-faint)] uppercase tracking-[0.1em] flex items-center gap-2">
              收货地址
            </label>
            <div class="relative">
              <MapPin class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--accent)]/60" />
              <input type="text" v-model="searchParams.address" placeholder="省、市、区、具体地址..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-[var(--text-secondary)]">
            </div>
          </div>

<!--          <div class="space-y-2">-->
<!--            <label class="text-[10px] font-bold text-[var(--text-faint)] uppercase tracking-[0.1em] flex items-center gap-2">-->
<!--              规格信息-->
<!--            </label>-->
<!--            <div class="relative">-->
<!--              <Package class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--accent)]/60" />-->
<!--              <input type="text" v-model="searchParams.item" placeholder="商品名称或规格..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-[var(--text-secondary)]">-->
<!--            </div>-->
<!--          </div>-->
          
          <div class="space-y-2 col-span-1 md:col-span-2">
            <label class="text-[10px] font-bold text-[var(--text-faint)] uppercase tracking-[0.1em] flex items-center gap-2">
              日期范围
            </label>
            <div class="relative">
              <ElDatePicker
                v-model="dateRange"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                class="w-full"
                :popper-class="'date-picker-popper'"
                :clearable="false"
                size="large"
                @change="handleDateChange"
              />
            </div>
          </div>

          <div class="space-y-2 ">
            <label class="text-[10px] font-bold text-[var(--text-faint)] uppercase tracking-[0.1em] flex items-center gap-2">
              关键词搜索
            </label>
            <div class="relative">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--accent)]/60" />
              <input type="text" v-model="searchParams.keyword" @input="handleKeywordSearch" placeholder="可搜姓名/手机号/地址/省份/商品/备注" class="w-full pl-9 pr-10 py-2.5 rounded-xl glass-input text-sm text-[var(--text-secondary)]">
              <button v-if="searchParams.keyword" @click="searchParams.keyword = ''" class="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-faint)] hover:text-[var(--text-primary)] transition-colors">
                &times;
              </button>
            </div>
          </div>

          <!-- 错误提示 -->
          <div v-if="errors.dateRange" class="col-span-full mt-2">
            <p class="text-red-400 text-xs font-medium">{{ errors.dateRange }}</p>
          </div>
        </div>
      </div>
      
      <div class="flex items-center justify-end gap-3 mt-6 pt-6 border-t border-[var(--border)]">
        <div class="flex gap-3">
        <button
            @click="exportToExcel"
            :disabled="orderDetails.length === 0"
            class="px-4 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] hover:text-[var(--text-primary)] font-bold border border-[var(--border-strong)] transition-all flex items-center gap-1.5 active:scale-95 text-xs">
          批量导出 <Download class="w-3.5 h-3.5" />
        </button>
        </div>
        <button @click="resetQuery" class="px-4 py-2 rounded-lg bg-[var(--bg-hover)] hover:bg-[var(--bg-hover-strong)] text-[var(--text-muted)] hover:text-[var(--text-primary)] font-bold border border-[var(--border-strong)] transition-all flex items-center gap-1.5 active:scale-95 text-xs">
          <RotateCcw class="w-3.5 h-3.5" /> 重置筛选
        </button>
        <button @click="fetchOrderData" class="px-6 py-2 rounded-lg bg-[var(--accent)] hover:bg-[var(--accent)] text-[var(--text-on-accent)] font-bold shadow-lg shadow-[var(--accent-glow)] transition-all flex items-center gap-1.5 active:scale-95 text-xs">
          <Search class="w-3.5 h-3.5" /> 立即查询
        </button>
      </div>
    </section>

    <!-- Province Map Section：省份订单分布（T19 懒加载，随上方筛选时间范围联动刷新） -->
    <section class="bg-[var(--bg-sidebar)] border border-[var(--border)] rounded-2xl p-4 shadow-sm">
      <div class="flex items-center gap-3 mb-4">
        <div class="w-1.5 h-6 bg-[var(--accent)] rounded-full"></div>
        <h3 class="text-lg font-bold font-display">省份订单分布</h3>
        <span class="text-xs text-[var(--text-faint)] font-medium tracking-wide">
          {{ provinceMapTimeRange.startTime ? '按筛选时间范围统计' : '按全部时间统计' }}
        </span>
      </div>
      <ProvinceMap :start-time="provinceMapTimeRange.startTime" :end-time="provinceMapTimeRange.endTime" />
    </section>

    <!-- Table Section -->
    <section class="bg-[var(--bg-sidebar)] border border-[var(--border)] rounded-2xl overflow-hidden shadow-xl relative">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm border-collapse">
          <thead class="bg-[var(--bg-hover)] text-[var(--text-muted)] font-medium">
            <tr class="border-b border-[var(--border)]">
              <th class="px-6 py-5">创建时间</th>
              <th class="px-6 py-5">客户姓名</th>
              <th class="px-6 py-5">联系电话</th>
              <th class="px-6 py-5">收货地址</th>
              <th class="px-6 py-5">省份</th>
              <th class="px-6 py-5">规格信息</th>
              <th class="px-6 py-5">备注</th>
<!--              <th class="px-6 py-5">数量</th>-->

              <th class="px-6 py-5">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)]">
            <tr v-for="(order, index) in orderDetails" :key="index" class="table-row-hover transition-colors" :class="{ 'bg-[var(--accent-soft-bg)]': index % 2 === 1 }">
              <td class="px-6 py-5 text-[var(--text-faint)] text-xs font-mono">{{ order.time }}</td>
              <td class="px-6 py-5 font-medium">{{ order.name || '未知姓名' }}</td>
              <td class="px-6 py-5 font-mono text-[var(--text-muted)]">{{ order.phone || '未知电话' }}</td>
              <td class="px-6 py-5 text-[var(--text-muted)] max-w-[200px]">{{ order.address || '未知地址' }}</td>
              <td class="px-6 py-5 text-[var(--text-muted)]">{{ order.province || '-' }}</td>

              <td class="px-6 py-5">
                <span class="px-2.5 py-1 rounded-md text-xs" :class="getTagClass(order.color)">
                  {{ order.item || '-' }}
                </span>
              </td>
              <td class="px-6 py-5 font-medium">{{ order.remark || '-' }}</td>
<!--              <td class="px-6 py-5 font-bold text-[var(--text-secondary)]">{{ order.quantity }}</td>-->

              <td class="px-6 py-5">
                <!-- 契约 §3.3：仅 id>0 的历史订单行显示操作按钮，预览行（id=0）不显示 -->
                <div v-if="order.id > 0" class="flex gap-2">
                  <button
                    class="p-2 rounded-lg bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                    title="编辑"
                    @click="openEditDialog(order)"
                  >
                    <Edit2 class="w-4 h-4" />
                  </button>
                  <button
                    class="p-2 rounded-lg bg-[var(--bg-hover)] text-red-400/60 hover:text-red-400 hover:bg-red-400/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    title="删除"
                    :disabled="deletingId === order.id"
                    @click="handleDeleteOrder(order)"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <!-- Pagination -->
      <div class="p-6 border-t border-[var(--border)] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div class="flex flex-col sm:flex-row items-start sm:items-center gap-4 w-full sm:w-auto">
          <!-- 每页显示条数选择器 -->
          <div class="flex items-center gap-2">
            <label class="text-xs text-[var(--text-faint)] font-medium">每页显示：</label>
            <select 
              v-model.number="pageSize" 
              @change="changePageSize(pageSize)"
              class="px-3 py-2 rounded-lg glass-input text-[var(--text-secondary)] text-xs transition-colors"
            >
              <option v-for="option in pageSizeOptions" :key="option" :value="option">{{ option }}</option>
            </select>
          </div>
          
          <!-- 显示信息 -->
          <p class="text-xs text-[var(--text-faint)] font-medium">
            显示 {{ (currentPage - 1) * pageSize + 1 }} 到 {{ Math.min(currentPage * pageSize, totalOrders) }} 条，共 {{ totalOrders }} 条记录
          </p>
        </div>
        
        <!-- 页码导航 -->
        <div class="flex items-center gap-2 w-full sm:w-auto justify-center sm:justify-end">
          <!-- 上一页按钮 -->
          <button 
            @click="prevPage"
            :disabled="currentPage === 1"
            class="w-8 h-8 flex items-center justify-center rounded-lg bg-[var(--bg-hover)] text-[var(--text-faint)] hover:text-[var(--text-primary)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="w-4 h-4" />
          </button>
          
          <!-- 页码按钮 -->
          <template v-for="(item, idx) in displayPages" :key="idx">
            <button
              v-if="item.type === 'page'"
              @click="changePage(item.value)"
              class="w-8 h-8 flex items-center justify-center rounded-lg text-xs font-bold transition-colors"
              :class="currentPage === item.value ? 'bg-[var(--accent)] text-[var(--text-on-accent)]' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text-on-accent)]'"
            >
              {{ item.value }}
            </button>
            <span v-else class="text-[var(--text-faint)] text-xs mx-1 px-1 select-none">...</span>
          </template>
          
          <!-- 下一页按钮 -->
          <button 
            @click="nextPage"
            :disabled="currentPage === totalPages"
            class="w-8 h-8 flex items-center justify-center rounded-lg bg-[var(--bg-hover)] text-[var(--text-faint)] hover:text-[var(--text-primary)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight class="w-4 h-4" />
          </button>
        </div>
      </div>
      
      <!-- 加载状态 -->
      <div v-if="isLoading" class="absolute inset-0 bg-black/50 flex items-center justify-center z-10">
        <div class="bg-[var(--bg-sidebar)] border border-[var(--border-strong)] rounded-2xl p-8 flex flex-col items-center gap-4">
          <div class="w-12 h-12 border-4 border-[var(--accent)] border-t-transparent rounded-full animate-spin"></div>
          <p class="text-[var(--text-secondary)] font-medium">加载中...</p>
        </div>
      </div>
    </section>

    <!-- 修改订单弹窗（T20） -->
    <EditOrderDialog
      :show="isEditDialogVisible"
      :order="editingOrder"
      @close="closeEditDialog"
      @saved="handleEditSaved"
    />
  </div>
</template>

<style scoped>
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
/* 日期选择框样式优化 */
input[type="date"] {
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
}
input[type="date"]::-webkit-calendar-picker-indicator {
  filter: invert(1) brightness(0.5);
  cursor: pointer;
}
input[type="date"]::-moz-calendar-picker-indicator {
  filter: invert(1) brightness(0.5);
  cursor: pointer;
}
/* 错误状态样式 */
input[type="date"].border-red-400 {
  border-color: #f87171;
  background-color: rgba(248, 113, 113, 0.05);
}

/* Element Plus 日期选择器样式 */
:deep(.el-date-editor) {
  width: 100% !important;
  min-width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box !important;
}

:deep(.el-date-editor .el-input__wrapper) {
  width: 100% !important;
  min-width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box !important;
}

:deep(.el-input__wrapper) {
  background-color: var(--bg-input);
  border: 1px solid var(--border-strong);
  border-radius: 0.75rem;
  transition: all 0.2s ease;
  box-sizing: border-box !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: var(--border-strong);
}

:deep(.el-input__wrapper.is-focus) {
  border-color: var(--accent);
  background-color: var(--bg-hover);
  box-shadow: 0 0 0 1px var(--accent) inset;
}

:deep(.el-input__inner) {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

:deep(.el-input__placeholder) {
  color: var(--text-muted);
}

:deep(.el-range-separator) {
  color: var(--text-muted);
}

/* 日期选择器弹出层样式 */
.date-picker-popper {
  background-color: var(--bg-card) !important;
  border: 1px solid var(--border-strong) !important;
  border-radius: 0.75rem !important;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3) !important;
}

:deep(.el-date-picker__header) {
  background-color: var(--bg-card) !important;
  border-bottom: 1px solid var(--border-strong) !important;
}

:deep(.el-date-picker__header-label) {
  color: var(--text-secondary) !important;
}

:deep(.el-date-picker__prev-btn:hover, .el-date-picker__next-btn:hover) {
  color: var(--accent) !important;
}

:deep(.el-date-table th) {
  color: var(--text-muted) !important;
}

:deep(.el-date-table td) {
  color: var(--text-secondary) !important;
}

:deep(.el-date-table td:hover) {
  background-color: var(--bg-hover) !important;
}

:deep(.el-date-table td.is-selected) {
  background-color: var(--accent) !important;
}

:deep(.el-date-table td.is-today) {
  color: var(--accent) !important;
}

:deep(.el-date-table td.is-disabled) {
  color: var(--text-faint) !important;
}

:deep(.el-picker-panel__footer) {
  background-color: var(--bg-card) !important;
  border-top: 1px solid var(--border-strong) !important;
}

:deep(.el-button--primary) {
  background-color: var(--accent) !important;
  border-color: var(--accent) !important;
}

:deep(.el-button--default) {
  background-color: var(--bg-hover) !important;
  border-color: var(--border-strong) !important;
  color: var(--text-secondary) !important;
}

:deep(.el-button--default:hover) {
  background-color: var(--bg-hover-strong) !important;
  border-color: var(--border-strong) !important;
}

.table-row-hover:hover {
  background-color: var(--bg-hover-faint);
}
.font-display {
  font-family: 'Cabinet Grotesk', sans-serif;
}
</style>

<style>
/* ElMessageBox 二次确认框暗色适配（T20）：
   该组件挂载于 body 之下，scoped/:deep 无法命中，需以命名空间类提供全局样式。 */
.order-action-msgbox {
  background-color: var(--bg-card) !important;
  border: 1px solid var(--border-strong) !important;
}
.order-action-msgbox .el-message-box__title {
  color: var(--text-secondary) !important;
}
.order-action-msgbox .el-message-box__message {
  color: var(--text-secondary) !important;
}
.order-action-msgbox .el-message-box__btns .el-button {
  background-color: var(--bg-hover) !important;
  border-color: var(--border-strong) !important;
  color: var(--text-secondary) !important;
}
.order-action-msgbox .el-message-box__btns .el-button--danger {
  background-color: #dc2626 !important;
  border-color: #dc2626 !important;
  color: #ffffff !important;
}
.order-action-msgbox .el-message-box__btns .el-button--primary {
  background-color: var(--accent) !important;
  border-color: var(--accent) !important;
  color: var(--text-on-accent) !important;
}
</style>
