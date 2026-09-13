<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
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
import { ElDatePicker } from 'element-plus';
import * as XLSX from "xlsx";

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

// Mock data for the table
const orderDetails = ref([
  { name: '张晓明', phone: '139-1234-5678', address: '浙江省杭州市西湖区文三路...', item: '高端人体工学椅', quantity: 1, time: '2023-11-20 14:32:11', color: 'blue' ,remark:'-'},
  { name: '李华', phone: '186-9988-7766', address: '广东省深圳市南山区高新园...', item: '机械键盘 (茶轴)', quantity: 2, time: '2023-11-20 15:05:44', color: 'teal',remark:'-' },
  { name: '王力群', phone: '155-0011-2233', address: '上海市浦东新区张江高科...', item: '4K 显示器 27寸', quantity: 1, time: '2023-11-21 09:12:05', color: 'blue',remark:'-' },
  { name: '陈思琪', phone: '177-4455-6677', address: '北京市朝阳区三里屯...', item: '降噪耳机 XM5', quantity: 1, time: '2023-11-21 11:45:30', color: 'purple',remark:'-' },
  { name: '赵雷', phone: '131-0099-8877', address: '四川省成都市武侯区...', item: '智能触控灯', quantity: 3, time: '2023-11-21 16:20:12', color: 'blue',remark:'-' },
]);

// 总订单数
const totalOrders = ref(12);

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
const displayPages = computed(() => {
  const total = totalPages.value;
  const current = currentPage.value;
  const pages: { type: 'page'; value: number } | { type: 'ellipsis'; value: number }[] = [];

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
  console.log('页码总数:', totalPages.value);
  console.log('切换页码:', page);
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
    const requestParams: any = {
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
    
    console.log('请求参数:', requestParams);
    
    // 调用后端订单查询接口
    const result = await orderApi.queryOrders(requestParams);
    console.log('查询结果:', result);
    
    // 检查返回值类型
    if (result && typeof result === 'object') {
      // 检查是否包含 list 和 count 字段
      if (Array.isArray(result.list) && (typeof result.count === 'number' || typeof result.count === 'string')) {

        // 更新总订单数
        totalOrders.value = Number(result.count);
        calculateTotalPages();
        
        // 根据返回的结果更新订单列表
        orderDetails.value = result.list.map((item: any) => ({
          name: item['收件人'] || '未知姓名',
          phone: item['手机号'] || '未知电话',
          address: item['收货地址'] || '未知地址',
          item: item['商品规格'] || '-',
          quantity: 1, // 后端返回的数据中可能没有数量字段，默认设为 1
          time: item.createdAt ? new Date(item.createdAt).toLocaleString() : '未知时间',
          color: 'blue', // 默认颜色为蓝色
          remark: item['备注'] || '-'
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
  isLoading.value = true;
  try {
    const addresssinfo = {
      "收件人": searchParams.value.name || '',
      "手机号": searchParams.value.phone || '',
      "收货地址": searchParams.value.address || '',
      "备注": ''
    };

    const requestParams: any = {
      limit: 0,   // limit=0 获取全部数据
      page: 1,
      key: searchParams.value.keyword || '',
      order: '',
      addresssinfo: addresssinfo
    };
    const startDate = new Date(searchParams.value.startTime);

    if (searchParams.value.startTime) {

      startDate.setHours(0, 0, 0, 0);
      requestParams.Starttime = startDate.toISOString();
    }
    const endDate = new Date(searchParams.value.endTime);
    if (searchParams.value.endTime) {
      endDate.setHours(23, 59, 59, 999);
      requestParams.Endtime = endDate.toISOString();
    }

    const result = await orderApi.queryOrders(requestParams);

    if (result && Array.isArray(result.list)) {
      const exportData = result.list.map((item: any) => ({
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
  } finally {
    isLoading.value = false;
  }
};


const getTagClass = (color: string) => {
  switch (color) {
    case 'blue': return 'bg-blue-500/10 text-blue-400';
    case 'teal': return 'bg-teal-500/10 text-teal-400';
    case 'purple': return 'bg-purple-500/10 text-purple-400';
    default: return 'bg-slate-500/10 text-slate-400';
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
  <div class="max-w-15xl mx-auto space-y-6">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <h2 class="text-3xl font-bold font-display">订单详情管理</h2>

    </div>

    <!-- Advanced Query Section -->
    <section class="bg-[#16191E] border border-white/5 rounded-2xl p-4 shadow-sm">
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="w-1.5 h-6 bg-blue-600 rounded-full"></div>
          <h3 class="text-lg font-bold font-display">高级查询筛选</h3>
        </div>
        <div class="flex items-center gap-3">
          <span class="text-xs text-slate-500 font-medium tracking-wide">共计 {{ totalOrders }} 条订单记录</span>
          <button 
            @click="toggleQuery"
            class="flex items-center gap-2 p-2 px-3 rounded-lg hover:bg-white/5 text-slate-400 hover:text-white transition-all text-xs font-medium border border-white/5 active:scale-95"
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
            <label class="text-[10px] font-bold text-slate-500 uppercase tracking-[0.1em] flex items-center gap-2">
              客户姓名
            </label>
            <div class="relative">
              <User class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-blue-500/60" />
              <input type="text" v-model="searchParams.name" placeholder="搜索姓名..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-slate-200">
            </div>
          </div>
          
          <div class="space-y-2">
            <label class="text-[10px] font-bold text-slate-500 uppercase tracking-[0.1em] flex items-center gap-2">
              联系电话
            </label>
            <div class="relative">
              <Phone class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-blue-500/60" />
              <input type="text" v-model="searchParams.phone" placeholder="手机号或固话..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-slate-200">
            </div>
          </div>
          
          <div class="space-y-2">
            <label class="text-[10px] font-bold text-slate-500 uppercase tracking-[0.1em] flex items-center gap-2">
              收货地址
            </label>
            <div class="relative">
              <MapPin class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-blue-500/60" />
              <input type="text" v-model="searchParams.address" placeholder="省、市、区、具体地址..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-slate-200">
            </div>
          </div>

<!--          <div class="space-y-2">-->
<!--            <label class="text-[10px] font-bold text-slate-500 uppercase tracking-[0.1em] flex items-center gap-2">-->
<!--              规格信息-->
<!--            </label>-->
<!--            <div class="relative">-->
<!--              <Package class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-blue-500/60" />-->
<!--              <input type="text" v-model="searchParams.item" placeholder="商品名称或规格..." class="w-full pl-9 pr-3 py-2.5 rounded-xl glass-input text-sm text-slate-200">-->
<!--            </div>-->
<!--          </div>-->
          
          <div class="space-y-2 col-span-1 md:col-span-2">
            <label class="text-[10px] font-bold text-slate-500 uppercase tracking-[0.1em] flex items-center gap-2">
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
            <label class="text-[10px] font-bold text-slate-500 uppercase tracking-[0.1em] flex items-center gap-2">
              关键词搜索
            </label>
            <div class="relative">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-blue-500/60" />
              <input type="text" v-model="searchParams.keyword" @input="handleKeywordSearch" placeholder="输入关键词搜索..." class="w-full pl-9 pr-10 py-2.5 rounded-xl glass-input text-sm text-slate-200">
              <button v-if="searchParams.keyword" @click="searchParams.keyword = ''" class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-white transition-colors">
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
      
      <div class="flex items-center justify-end gap-3 mt-6 pt-6 border-t border-white/5">
        <div class="flex gap-3">
        <button
            @click="exportToExcel"
            :disabled="orderDetails.length === 0"
            class="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white font-bold border border-white/10 transition-all flex items-center gap-1.5 active:scale-95 text-xs">
          批量导出 <Download class="w-3.5 h-3.5" />
        </button>
        </div>
        <button @click="resetQuery" class="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white font-bold border border-white/10 transition-all flex items-center gap-1.5 active:scale-95 text-xs">
          <RotateCcw class="w-3.5 h-3.5" /> 重置筛选
        </button>
        <button @click="fetchOrderData" class="px-6 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-bold shadow-lg shadow-blue-600/20 transition-all flex items-center gap-1.5 active:scale-95 text-xs">
          <Search class="w-3.5 h-3.5" /> 立即查询
        </button>
      </div>
    </section>

    <!-- Table Section -->
    <section class="bg-[#16191E] border border-white/5 rounded-2xl overflow-hidden shadow-xl relative">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm border-collapse">
          <thead class="bg-white/5 text-slate-400 font-medium">
            <tr class="border-b border-white/5">
              <th class="px-6 py-5">创建时间</th>
              <th class="px-6 py-5">客户姓名</th>
              <th class="px-6 py-5">联系电话</th>
              <th class="px-6 py-5">收货地址</th>
              <th class="px-6 py-5">规格信息</th>
              <th class="px-6 py-5">备注</th>
<!--              <th class="px-6 py-5">数量</th>-->

<!--              <th class="px-6 py-5">操作</th>-->
            </tr>
          </thead>
          <tbody class="divide-y divide-white/5">
            <tr v-for="(order, index) in orderDetails" :key="index" class="table-row-hover transition-colors" :class="{ 'bg-blue-600/[0.04]': index % 2 === 1 }">
              <td class="px-6 py-5 text-slate-500 text-xs font-mono">{{ order.time }}</td>
              <td class="px-6 py-5 font-medium">{{ order.name }}</td>
              <td class="px-6 py-5 font-mono text-slate-400">{{ order.phone }}</td>
              <td class="px-6 py-5 text-slate-400 max-w-[200px]">{{ order.address }}</td>

              <td class="px-6 py-5">
                <span class="px-2.5 py-1 rounded-md text-xs" :class="getTagClass(order.color)">
                  {{ order.item }}
                </span>
              </td>
              <td class="px-6 py-5 font-medium">{{ order.remark }}</td>
<!--              <td class="px-6 py-5 font-bold text-slate-200">{{ order.quantity }}</td>-->

<!--              <td class="px-6 py-5">-->
<!--                <div class="flex gap-2">-->
<!--                  <button class="p-2 rounded-lg bg-white/5 text-slate-400 hover:text-white transition-colors" title="编辑">-->
<!--                    <Edit2 class="w-4 h-4" />-->
<!--                  </button>-->
<!--                  <button class="p-2 rounded-lg bg-white/5 text-red-400/60 hover:text-red-400 hover:bg-red-400/10 transition-colors" title="删除">-->
<!--                    <Trash2 class="w-4 h-4" />-->
<!--                  </button>-->
<!--                </div>-->
<!--              </td>-->
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
            显示 {{ (currentPage - 1) * pageSize + 1 }} 到 {{ Math.min(currentPage * pageSize, totalOrders) }} 条，共 {{ totalOrders }} 条记录
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
            <ChevronLeft class="w-4 h-4" />
          </button>
          
          <!-- 页码按钮 -->
          <template v-for="(item, idx) in displayPages" :key="idx">
            <button
              v-if="item.type === 'page'"
              @click="changePage(item.value)"
              class="w-8 h-8 flex items-center justify-center rounded-lg text-xs font-bold transition-colors"
              :class="currentPage === item.value ? 'bg-blue-600 text-white' : 'bg-white/5 text-slate-400 hover:text-white'"
            >
              {{ item.value }}
            </button>
            <span v-else class="text-slate-600 text-xs mx-1 px-1 select-none">...</span>
          </template>
          
          <!-- 下一页按钮 -->
          <button 
            @click="nextPage"
            :disabled="currentPage === totalPages"
            class="w-8 h-8 flex items-center justify-center rounded-lg bg-white/5 text-slate-500 hover:text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight class="w-4 h-4" />
          </button>
        </div>
      </div>
      
      <!-- 加载状态 -->
      <div v-if="isLoading" class="absolute inset-0 bg-black/50 flex items-center justify-center z-10">
        <div class="bg-[#16191E] border border-white/10 rounded-2xl p-8 flex flex-col items-center gap-4">
          <div class="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
          <p class="text-slate-300 font-medium">加载中...</p>
        </div>
      </div>
    </section>
  </div>
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
  background-color: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.75rem;
  transition: all 0.2s ease;
  box-sizing: border-box !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: rgba(255, 255, 255, 0.2);
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #3B82F6;
  background-color: rgba(255, 255, 255, 0.06);
  box-shadow: 0 0 0 1px #3B82F6 inset;
}

:deep(.el-input__inner) {
  color: #e2e8f0;
  font-size: 0.875rem;
}

:deep(.el-input__placeholder) {
  color: #94A3B8;
}

:deep(.el-range-separator) {
  color: #94A3B8;
}

/* 日期选择器弹出层样式 */
.date-picker-popper {
  background-color: #1E2228 !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
  border-radius: 0.75rem !important;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3) !important;
}

:deep(.el-date-picker__header) {
  background-color: #1E2228 !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1) !important;
}

:deep(.el-date-picker__header-label) {
  color: #e2e8f0 !important;
}

:deep(.el-date-picker__prev-btn:hover, .el-date-picker__next-btn:hover) {
  color: #3B82F6 !important;
}

:deep(.el-date-table th) {
  color: #94A3B8 !important;
}

:deep(.el-date-table td) {
  color: #e2e8f0 !important;
}

:deep(.el-date-table td:hover) {
  background-color: rgba(255, 255, 255, 0.05) !important;
}

:deep(.el-date-table td.is-selected) {
  background-color: #3B82F6 !important;
}

:deep(.el-date-table td.is-today) {
  color: #3B82F6 !important;
}

:deep(.el-date-table td.is-disabled) {
  color: #475569 !important;
}

:deep(.el-picker-panel__footer) {
  background-color: #1E2228 !important;
  border-top: 1px solid rgba(255, 255, 255, 0.1) !important;
}

:deep(.el-button--primary) {
  background-color: #3B82F6 !important;
  border-color: #3B82F6 !important;
}

:deep(.el-button--default) {
  background-color: rgba(255, 255, 255, 0.05) !important;
  border-color: rgba(255, 255, 255, 0.1) !important;
  color: #e2e8f0 !important;
}

:deep(.el-button--default:hover) {
  background-color: rgba(255, 255, 255, 0.1) !important;
  border-color: rgba(255, 255, 255, 0.2) !important;
}

.table-row-hover:hover {
  background-color: rgba(255, 255, 255, 0.02);
}
.font-display {
  font-family: 'Cabinet Grotesk', sans-serif;
}
</style>
