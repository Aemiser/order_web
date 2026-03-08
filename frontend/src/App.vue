<script setup lang="ts">
import { ref, computed } from 'vue';
import { Info, Download, RotateCw, SlidersHorizontal, Wallet } from 'lucide-vue-next';
import * as XLSX from 'xlsx';

import Sidebar from './components/Sidebar/Sidebar.vue';
import demo from './components/demo.vue';
import Header from './components/Header/Header.vue';
import FieldDialog from './components/Main/Dialog/Field.vue';
import OrderInputView from './components/Main/OrderInputView.vue';
import OrderDetailsView from './components/Main/OrderDetailsView.vue';
import AppSettingsView from './components/Main/AppSettingsView.vue';
import CheckoutView from './components/Main/Checkout/CheckoutView.vue';
import { OrderRecognition, GetpreviewData, DeletepreviewData, CreateOrder } from '../wailsjs/go/main/App';
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
  showFieldDialog.value = true;
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
            <span>已选定 8 天数据</span>
          </div>
          <div class="w-px h-4 bg-white/10"></div>
          <p>总成交品种: <span class="text-white font-bold">14项</span></p>
        </div>
        <div class="flex items-center gap-3">
          <button 
            class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white transition-all flex items-center gap-2"
          >
            <RotateCw class="w-4 h-4" />
            <span class="text-xs font-bold">刷新数据</span>
          </button>
          <button 
            @click="handleConfig"
            class="h-10 px-5 rounded-xl border border-white/10 text-slate-400 hover:bg-white/5 hover:text-white transition-all flex items-center gap-2"
          >
            <SlidersHorizontal class="w-4 h-4" />
            <span class="text-xs font-bold">字段配置</span>
          </button>
          <button 
            class="h-10 px-8 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-bold shadow-lg shadow-blue-600/20 transition-all flex items-center gap-2"
          >
            <Wallet class="w-4 h-4" />
            <span class="text-xs">结算账单</span>
          </button>
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
        :fieldList="[]"
        :pbMap="{}"
        @close="showFieldDialog = false" 
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
