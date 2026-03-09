<script setup lang="ts">
import { ref, computed } from 'vue';
import { Info, Download, RotateCw, SlidersHorizontal, Wallet, DollarSign } from 'lucide-vue-next';
import * as XLSX from 'xlsx';

import Sidebar from './components/Sidebar/Sidebar.vue';
import demo from './components/demo.vue';
import Header from './components/Header/Header.vue';
import FieldDialog from './components/Main/Dialog/Field.vue';

import OrderInputView from './components/Main/OrderInputView.vue';
import OrderDetailsView from './components/Main/OrderDetailsView.vue';
import AppSettingsView from './components/Main/AppSettingsView.vue';
import CheckoutView from './components/Main/Checkout/CheckoutView.vue';
import { OrderRecognition, GetpreviewData, DeletepreviewData, CreateOrder, SavePriceData, GetDaydata } from '../wailsjs/go/main/App';
import type { Order } from './types';


// --- State ---
const activeTab = ref('input');
const inputText = ref('');
const orders = ref<Order[]>([]);
const isAnalyzing = ref(false);
const isSystemReady = ref(true);
const statusMessage = ref('系统已就绪 (后端模式)');
const showConfirmModal = ref(false);
const showFieldDialog = ref(false);
const isEditMode = ref(false);
const checkoutResponse = ref<any>(null);
const isLoading = ref(false);
const proPriceRequests = ref<any[]>([]);
const hasCheckoutData = ref(true); // 用于跟踪结账表格是否有数据
const selectedDateRange = ref<{start: string, end: string}>({
  start: '2026-03-01',
  end: '2026-03-31'
});

// 字段数据
const fieldData = ref<{
  fieldList: string[];
  pbMap: Record<string, string[]>
}>({
  fieldList: [],
  pbMap: {}
});

// 数据加载状态
const isLoadingFieldData = ref(false);
const fieldDataLoaded = ref(false);

// --- Logic ---
const analyzeOrders = async (isAppend: boolean = false) => {
  if (!inputText.value.trim()) return;
  
  isAnalyzing.value = true;
  statusMessage.value = '正在识别订单数据...';
  
  try {
    // 调用后端的OrderRecognition方法，第二个参数为模式：0-重来，1-追加
    const model = isAppend ? 1 : 0;
    const result = await OrderRecognition(inputText.value, model);
    console.log(result);
    // 转换后端返回的数据为Order类型
    const newOrders: Order[] = [];
    
    if (Array.isArray(result)) {
      orders.value = [];
      result.forEach((item: any, index: number) => {
        
        newOrders.push({
          id: item.PreviewID || item['VID'],
          name: item.Recipient || item['收件人'] || '未知姓名',
          phone: item.Phone || item['手机号'] || '未知电话',
          address: item.Address || item['收货地址'] || '未知地址',
          items: item.ProductSpecs || item['商品规格'] || '-',
          note: item.Remarks || item['备注'] || '-'
        });
      });
    }

    if (newOrders.length > 0) {
      orders.value = [...orders.value, ...newOrders];
      inputText.value = '';
      statusMessage.value = `成功解析 ${newOrders.length} 条订单`;
    } else {
      statusMessage.value = '解析失败：格式不匹配';
    }
    
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error("Error calling backend:", error);
    statusMessage.value = '解析出错：后端服务异常';
  } finally {
    isAnalyzing.value = false;
  }
};

const clearInput = () => {
  inputText.value = '';
};

const removeOrder = (id: string) => {
  orders.value = orders.value.filter(o => o.id !== id);
};

const deleteOrder = async (id: string) => {
  try {
    // 调用后端的 DeletepreviewData 方法
    const result = await DeletepreviewData(parseInt(id));
    statusMessage.value = result;
    
    // 刷新数据
    await refreshData();
    
    // 3秒后恢复默认状态
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error('删除失败:', error);
    statusMessage.value = '删除出错：后端服务异常';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  }
};

const exportToExcel = () => {
  // 定义字段映射关系
  const fieldMap = {
    name: '收件人',
    phone: '手机号',
    address: '收货地址',
    items: '规格信息',
    note: '备注'
  };

  // 转换数据：只保留需要的字段并重命名
  const exportData = orders.value.map(order => {
    const newRow = {};
    Object.keys(fieldMap).forEach(key => {
      if (order.hasOwnProperty(key)) {
        newRow[fieldMap[key]] = order[key];
      }
    });
    return newRow;
  });

  const worksheet = XLSX.utils.json_to_sheet(exportData);
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, "Orders");
  XLSX.writeFile(workbook, `Orders_${new Date().toISOString().split('T')[0]}.xlsx`);
};

const refreshData = async () => {
  try {
    const result = await GetpreviewData();
    if (Array.isArray(result)) {
      const newOrders: Order[] = result.map((item: any) => ({
        id: item.VID,
        name: item['收件人'] || '未知姓名',
        phone: item['手机号'] || '未知电话',
        address: item['收货地址'] || '未知地址',
        items: item['商品规格'] || '-',
        note: item['备注'] || '-'
      }));
      orders.value = newOrders;
    } else {
      // 如果返回的数据不是数组，将 orders 设置为空数组
      orders.value = [];
    }
  } catch (error) {
    console.error('刷新数据失败:', error);
    // 发生错误时，也将 orders 设置为空数组
    orders.value = [];
  }
};

const totalOrders = computed(() => orders.value.length);
const estimatedSize = computed(() => (orders.value.length * 4.1).toFixed(1));

// 计算选中的天数
const selectedDays = computed(() => {
  if (!selectedDateRange.value.start || !selectedDateRange.value.end) return 0;
  
  const startDate = new Date(selectedDateRange.value.start);
  const endDate = new Date(selectedDateRange.value.end);
  
  // 计算时间差（毫秒）
  const timeDiff = endDate.getTime() - startDate.getTime();
  
  // 转换为天数并取整
  const days = Math.ceil(timeDiff / (1000 * 3600 * 24)) + 1; // +1 因为包含开始和结束日期
  
  return days;
});

const createOrder = () => {
  // 显示确认模态弹窗
  showConfirmModal.value = true;
};

const handleConfirm = async () => {
  // 关闭模态弹窗
  showConfirmModal.value = false;
  
  try {
    const result = await CreateOrder();
    statusMessage.value = result;
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error('创建订单失败:', error);
    statusMessage.value = '创建订单出错：后端服务异常';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  }
};

const handleCancel = () => {
  // 关闭模态弹窗
  showConfirmModal.value = false;
};

// 处理字段配置
const handleConfig = () => {
  // 检查是否有字段数据
  if (fieldData.value.fieldList.length === 0 && Object.keys(fieldData.value.pbMap).length === 0) {
    // 如果没有字段数据，显示提示
    alert('请先执行查询以获取字段数据');
    return;
  }
  showFieldDialog.value = true;
};

// 处理保存字段配置
const handleSaveFieldConfig = (fieldMap: Record<string, string[]>) => {
  console.log('保存字段配置:', fieldMap);
  
  try {
    // 调用后端 SetProductBelong 函数
    window['go']['main']['App']['SetProductBelong'](fieldMap);
    console.log('字段配置保存成功');
    statusMessage.value = '字段配置保存成功';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error('保存字段配置失败:', error);
    alert('保存字段配置失败，请稍后重试');
  }
  
  showFieldDialog.value = false;
};

// 处理进入编辑模式
const handleEnterEditMode = () => {
  isEditMode.value = true;
  
  // 初始化 proPriceRequests 状态
  if (checkoutResponse.value && checkoutResponse.value.ProSPecs) {
    const proPriceRequestsData = [];
    let specIndex = 0;
    
    checkoutResponse.value.ProSPecs.forEach((product: any) => {
      if (product.Specs) {
        product.Specs.forEach((spec: any) => {
          const pricePerItem = checkoutResponse.value.PricePerItemRow ? checkoutResponse.value.PricePerItemRow[specIndex] : 0;
          const pricePerKg = checkoutResponse.value.PricePerKgRow ? checkoutResponse.value.PricePerKgRow[specIndex] : 0;
          
          proPriceRequestsData.push({
            productName: product.Name || '',
            spec: typeof spec === 'object' ? (spec.name || spec.Name || JSON.stringify(spec)) : spec,
            total: 0, // 后端会计算
            pricePerKg: typeof pricePerKg === 'number' ? pricePerKg : 0,
            pricePerItem: typeof pricePerItem === 'number' ? pricePerItem : 0,
            totalAmount: 0 // 后端会计算
          });
          
          specIndex++;
        });
      }
    });
    
    proPriceRequests.value = proPriceRequestsData;
  } else {
    // 如果没有后端数据，使用默认商品数据初始化
    const defaultProducts = [
      { name: '香水柠檬', specs: ['5斤', '8斤', '10斤', '20斤'] },
      { name: '广香一级', specs: ['10斤', '20斤'] },
      { name: '广香二级', specs: ['7斤', '15斤'] },
      { name: '黄柠檬', specs: ['3斤', '6斤'] },
      { name: '金桔', specs: ['2斤', '4斤'] }
    ];
    
    const proPriceRequestsData = [];
    
    defaultProducts.forEach(product => {
      product.specs.forEach(spec => {
        proPriceRequestsData.push({
          productName: product.name,
          spec: spec,
          total: 0, // 后端会计算
          pricePerKg: 0,
          pricePerItem: 0,
          totalAmount: 0 // 后端会计算
        });
      });
    });
    
    proPriceRequests.value = proPriceRequestsData;
  }
};

// 处理退出编辑模式
const handleExitEditMode = () => {
  isEditMode.value = false;
};

// 处理保存价格
const handleSavePrices = async (data?: any[]) => {
  try {
    // 如果有数据传入，且数据是数组，存储到状态中
    if (data && Array.isArray(data)) {
      proPriceRequests.value = data;
    }
    
    // 验证 proPriceRequests 是否为空
    if (!proPriceRequests.value || proPriceRequests.value.length === 0) {
      statusMessage.value = '没有可保存的价格数据';
      setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
      return;
    }
    
    // 显示加载状态
    isLoading.value = true;
    statusMessage.value = '正在保存价格数据...';
    
    // 调用后端的 SetPriceData 方法
    const response = await window['go']['main']['App']['SetPriceData'](proPriceRequests.value);
    console.log('保存价格成功:', response);
    // 存储后端响应数据
    checkoutResponse.value = response;
    statusMessage.value = '价格保存成功';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
    isEditMode.value = false;
  } catch (error) {
    console.error('保存价格失败:', error);
    statusMessage.value = '保存价格失败，请重试';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } finally {
    // 隐藏加载状态
    isLoading.value = false;
  }
};

// 处理保存计量单位
const handleSaveUnit = (unit: string) => {
  console.log('保存计量单位:', unit);
  // 这里可以添加保存计量单位的逻辑
  // 例如存储到本地存储或发送到后端
  statusMessage.value = '计量单位保存成功';
  setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
};

// 处理清空所有价格输入
const handleClearPrices = () => {
  // 清空价格请求数据
  proPriceRequests.value = [];
  statusMessage.value = '价格数据已清空';
  setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
};

// 处理结账表格数据状态变化
const handleCheckoutDataStatus = (hasData: boolean) => {
  hasCheckoutData.value = hasData;
};

// 处理结算账单
const handleCheckout = async () => {
  try {
    // 显示加载状态
    isLoading.value = true;
    statusMessage.value = '正在生成财务账单...';
    
    // 调用后端的 SaveCheackToExcel 方法
    const response = await window['go']['main']['App']['SaveCheackToExcel']();
    console.log('结算账单成功:', response);
    statusMessage.value = '财务账单生成成功';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } catch (error) {
    console.error('结算账单失败:', error);
    statusMessage.value = '生成财务账单失败，请重试';
    setTimeout(() => statusMessage.value = '系统已就绪 (后端模式)', 3000);
  } finally {
    // 隐藏加载状态
    isLoading.value = false;
  }
};

// 处理日期范围变化
const handleDateRangeChange = (dateRange: {start: string, end: string}) => {
  selectedDateRange.value = dateRange;
  console.log('日期范围已更新:', dateRange);
};

// 处理字段数据更新
const handleUpdateFieldData = (data: {
  fieldList: string[];
  pbMap: Record<string, string[]>;
}) => {
  fieldData.value = data;
  console.log('字段数据已更新:', data);
  
  // 检查PBMap是否有效
  fieldDataLoaded.value = !!data && !!data.pbMap && Object.keys(data.pbMap).length > 0;
  isLoadingFieldData.value = false;
};

// 处理字段数据加载状态
const handleFieldDataLoading = (isLoading: boolean) => {
  isLoadingFieldData.value = isLoading;
  console.log('字段数据加载状态:', isLoading);
};



</script>

<template>
  <div class="flex h-screen w-full overflow-hidden bg-[#0F1115] text-white font-sans">
    <Sidebar
      v-model:activeTab="activeTab"
      :isSystemReady="isSystemReady"
      :statusMessage="statusMessage"
    />

    <main class="flex-1 flex flex-col min-w-0 bg-[#0F1115]">
      <Header
        userName="Admin User"
        userRole="高级管理员"
      />

      <div class="flex-1 overflow-y-auto p-8 custom-scrollbar">
        <!-- Conditional View Rendering -->
        <OrderInputView
          v-if="activeTab === 'input'"
          v-model:inputText="inputText"
          :isAnalyzing="isAnalyzing"
          :orders="orders"
          @clear="clearInput"
          @analyze="analyzeOrders"
          @removeOrder="removeOrder"
          @clearAllOrders="orders = []"
          @refreshData="refreshData"
          @delete="deleteOrder"
        />

        <OrderDetailsView
          v-if="activeTab === 'details'"
          :activeTab="activeTab"
        />

        <CheckoutView
          v-if="activeTab === 'checkout'"
          :isEditMode="isEditMode"
          :checkoutResponse="checkoutResponse"
          @save-prices="handleSavePrices"
          @save-unit="handleSaveUnit"
          @enter-edit-mode="handleEnterEditMode"
          @exit-edit-mode="handleExitEditMode"
          @clear-prices="handleClearPrices"
          @data-status="handleCheckoutDataStatus"
          @date-range-change="handleDateRangeChange"
          @update-field-data="handleUpdateFieldData"
          @field-data-loading="handleFieldDataLoading"
        />

        <AppSettingsView
          v-if="activeTab === 'settings'"
        />

       <!-- <demo />
        <Fidld /> -->
      </div>

      <!-- Bottom Action Bar (Only show for Input view) -->
      <footer 
        v-if="activeTab === 'input'"
        class="h-20 bg-[#16191E] border-t border-white/5 px-8 flex items-center justify-between shrink-0"
      >
        <div class="flex items-center gap-4 text-sm text-slate-400">
          <span class="flex items-center gap-2">
            <Info class="w-4 h-4 text-blue-500" />
            共计识别 {{ totalOrders }} 条订单，预计文件大小 {{ estimatedSize }} KB
          </span>
        </div>
        <div class="flex items-center gap-4">
          <button @click="createOrder" class="px-8 py-3 rounded-xl bg-white/5 hover:bg-white/10 text-white font-medium transition-all">
            确认数据无误
          </button>
          <button 
            @click="exportToExcel"
            :disabled="orders.length === 0"
            class="px-10 py-3 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold flex items-center gap-2 shadow-xl shadow-blue-600/30 transition-all"
          >
            导出 XLSX 表格 <Download class="w-4 h-4" />
          </button>
        </div>
      </footer>

      <!-- Bottom Action Bar (Only show for Checkout view) -->
      <footer 
        v-if="activeTab === 'checkout'"
        class="h-20 bg-[#16191E] border-t border-white/5 px-8 flex items-center justify-between shrink-0"
      >
        <div class="flex items-center gap-4 text-xs text-slate-500">
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-blue-600"></span>
            <span>已选定 {{ selectedDays }} 天数据</span>
          </div>
          <div class="w-px h-4 bg-white/10"></div>
          <p>总成交品种: <span class="text-white font-bold">14项</span></p>
        </div>
        <div class="flex items-center gap-3">
          <!-- 编辑模式下的按钮 -->
          <template v-if="isEditMode">
            <button 
              @click="handleClearPrices"
              class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white transition-all flex items-center gap-2"
            >
              <span class="text-xs font-bold">清空</span>
            </button>
            <button 
              @click="handleExitEditMode"
              class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white transition-all flex items-center gap-2"
            >
              <span class="text-xs font-bold">取消</span>
            </button>
            <button 
              @click="handleSavePrices"
              :disabled="isLoading"
              class="h-10 px-8 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold shadow-lg shadow-blue-600/20 transition-all flex items-center gap-2"
            >
              <span v-if="isLoading" class="animate-spin mr-2">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
                </svg>
              </span>
              <span class="text-xs">{{ isLoading ? '保存中...' : '确定' }}</span>
            </button>
          </template>
          <!-- 非编辑模式下的按钮 -->
          <template v-else>
            <button 
              :disabled="!hasCheckoutData"
              class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center gap-2"
            >
              <RotateCw class="w-4 h-4" />
              <span class="text-xs font-bold">刷新数据</span>
            </button>
            <button 
              @click="handleConfig"
              :disabled="!fieldDataLoaded || isLoadingFieldData"
              class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white transition-all flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div v-if="isLoadingFieldData" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
              <SlidersHorizontal v-else class="w-4 h-4" />
              <span class="text-xs font-bold">{{ isLoadingFieldData ? '加载中...' : '字段配置' }}</span>
            </button>
            <button 
              @click="handleEnterEditMode"
              :disabled="!hasCheckoutData"
              class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center gap-2"
            >
              <DollarSign class="w-4 h-4" />
              <span class="text-xs font-bold">编辑金额</span>
            </button>
            <button 
              @click="handleCheckout"
              :disabled="!hasCheckoutData"
              class="h-10 px-8 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-bold shadow-lg shadow-blue-600/20 transition-all flex items-center gap-2"
            >
              <Wallet class="w-4 h-4" />
              <span class="text-xs font-bold">财务结算</span>
            </button>
          </template>
        </div>
      </footer>

      <!-- Confirmation Modal -->
      <div v-if="showConfirmModal" class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 transition-opacity duration-300">
        <div class="bg-[#16191E] border border-white/10 rounded-2xl p-8 max-w-md w-full mx-4 transform transition-all duration-300 scale-100">
          <h3 class="text-xl font-bold text-white mb-4 font-display">确认操作</h3>
          <p class="text-slate-300 mb-6">请确保今日的订单绝对正确，不然会影响结账！！！</p>
          <div class="flex justify-end gap-4">
            <button @click="handleCancel" class="px-6 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white font-medium transition-all">
              取消
            </button>
            <button @click="handleConfirm" class="px-6 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium transition-all">
              确定
            </button>
          </div>
        </div>
      </div>

      <!-- Field Configuration Dialog -->
      <FieldDialog 
        v-if="showFieldDialog" 
        :show="showFieldDialog" 
        :fieldList="fieldData.fieldList"
        :pbMap="fieldData.pbMap"
        @close="showFieldDialog = false" 
        @save="handleSaveFieldConfig"
      />
      

    </main>
  </div>
</template>

<style>
@import url('https://api.fontshare.com/v2/css?f[]=satoshi@700,500,400&f[]=cabinet-grotesk@800&display=swap');

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #334155;
  border-radius: 10px;
}
</style>
