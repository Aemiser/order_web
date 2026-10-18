<template>
  <div class="flex-1 overflow-y-auto p-6 custom-scrollbar">
    <!-- Query Area -->
    <CheckQuery 
      @query="handleQuery"
      @generate-test-data="handleGenerateTestData"
      @show-field-dialog="handleShowFieldDialog"
      @update-field-data="handleUpdateFieldData"
      @field-data-loading="handleFieldDataLoading"
      :initialDateStart="initialDateStart"
      :initialDateEnd="initialDateEnd"
    />

    <!-- Table Area -->
    <CheckTable 
      :testData="testData" 
      :checkoutResponse="props.checkoutResponse"
      :isEditMode="props.isEditMode" 
      @save-prices="handleSavePrices"
      @save-unit="handleSaveUnit"
      @enter-edit-mode="handleEnterEditMode"
      @exit-edit-mode="handleExitEditMode"
      @clear-prices="handleClearPrices"
      @data-status="handleDataStatus"
    />

    <!-- Field Configuration Dialog -->
    <FieldDialog 
      v-if="showFieldDialog" 
      :show="showFieldDialog"
      :fieldList="fieldList"
      :pbMap="pbMap"
      @close="handleCloseFieldDialog"
      @save="handleSaveFieldConfig"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { RotateCw, SlidersHorizontal, Wallet } from 'lucide-vue-next';
import CheckQuery from './CheckQuery.vue';
import CheckTable from './CheckTable.vue';
import FieldDialog from '../Dialog/Field.vue';
import * as checkoutApi from '../../../api/checkout';

// 初始日期范围
const initialDateStart = ref('2024-02-01');
const initialDateEnd = ref('2024-02-10');

// 字段配置弹窗显示状态
const showFieldDialog = ref(false);

// 测试数据状态
const testData = ref<any>(null);

// 字段列表和商品归属map
const fieldList = ref<string[]>([]);
const pbMap = ref<Record<string, string[]>>({});

// 存储从CheckQuery组件获取的字段数据
const fieldData = ref<{
  fieldList: string[];
  pbMap: Record<string, string[]>
}>({
  fieldList: [],
  pbMap: {}
});

// 编辑模式状态
const props = defineProps<{
  isEditMode?: boolean;
  checkoutResponse?: any;
}>();

// 定义事件
const emit = defineEmits<{
  (e: 'save-prices', prices: (number | string)[]): void;
  (e: 'save-unit', unit: string): void;
  (e: 'enter-edit-mode'): void;
  (e: 'exit-edit-mode'): void;
  (e: 'clear-prices'): void;
  (e: 'data-status', hasData: boolean): void;
  (e: 'date-range-change', dateRange: {start: string, end: string}): void;
  (e: 'update-field-data', data: {
    fieldList: string[];
    pbMap: Record<string, string[]>;
  }): void;
  (e: 'field-data-loading', isLoading: boolean): void;
}>();

// 处理保存价格
const handleSavePrices = (proPriceRequests: any[]) => {
  emit('save-prices', proPriceRequests);
};

// 处理保存计量单位
const handleSaveUnit = (unit: string) => {
  emit('save-unit', unit);
};

// 处理进入编辑模式
const handleEnterEditMode = () => {
  emit('enter-edit-mode');
};

// 处理退出编辑模式
const handleExitEditMode = () => {
  emit('exit-edit-mode');
};

// 处理清空价格
const handleClearPrices = () => {
  emit('clear-prices');
};

// 处理数据状态变化
const handleDataStatus = (hasData: boolean) => {
  emit('data-status', hasData);
};

// 处理查询
const handleQuery = (params: {
  dateStart: string;
  dateEnd: string;
  category: string;
  keyword: string;
  data?: any;
}) => {
  console.log('查询参数:', params);
  
  // 传递日期范围给父组件
  emit('date-range-change', {
    start: params.dateStart,
    end: params.dateEnd
  });
  
  // 确保数据结构完整
  if (params.data) {
    // 验证数据结构
    const validData = {
      FieldList: params.data.FieldList || [],
      Count: params.data.Count || 0,
      ProSPecs: params.data.ProSPecs || [],
      TableData: params.data.TableData || [],
      PBMap: params.data.PBMap || {},
      TotalRow: params.data.TotalRow || [],
      PricePerKgRow: params.data.PricePerKgRow || [],
      PricePerItemRow: params.data.PricePerItemRow || [],
      totalAmountRow: params.data.totalAmountRow || [],
    };
    
    // 更新测试数据，触发 CheckTable 组件的更新
    testData.value = validData;
    console.log('数据已传递给 CheckTable 组件');
  }
};

// 处理测试数据生成
const handleGenerateTestData = (data: any) => {
  console.log('收到测试数据:', data);
  testData.value = data;
};

// 处理显示字段配置弹窗
const handleShowFieldDialog = (data: {
  fieldList: string[];
  pbMap: Record<string, string[]>;
}) => {
  console.log('显示字段配置弹窗:', data);
  fieldList.value = data.fieldList;
  
  pbMap.value = data.pbMap;
  
  showFieldDialog.value = true;
};

// 处理关闭字段配置弹窗
const handleCloseFieldDialog = () => {
  showFieldDialog.value = false;
};

// 处理保存字段配置
const handleSaveFieldConfig = async (fieldMap: Record<string, string[]>) => {
  console.log('保存字段配置:', fieldMap);

  try {
    await checkoutApi.setProductBelong(fieldMap);
    console.log('字段配置保存成功');
  } catch (error) {
    console.error('保存字段配置失败:', error);
    alert('保存字段配置失败，请稍后重试');
  }

  showFieldDialog.value = false;
};

// 处理刷新
const handleRefresh = () => {
  console.log('刷新数据');
  // 这里可以添加实际的刷新逻辑
};

// 处理字段数据更新
const handleUpdateFieldData = (data: {
  fieldList: string[];
  pbMap: Record<string, string[]>;
}) => {
  // 存储字段数据
  fieldData.value = data;
  // 传递给父组件
  emit('update-field-data', data);
  console.log('字段数据已更新:', data);
};

// 处理字段数据加载状态
const handleFieldDataLoading = (isLoading: boolean) => {
  // 传递给父组件
  emit('field-data-loading', isLoading);
  console.log('字段数据加载状态:', isLoading);
};

// 处理字段配置
const handleConfig = () => {
  // 使用存储的字段数据显示弹窗
  fieldList.value = fieldData.value.fieldList;
  pbMap.value = fieldData.value.pbMap;
  showFieldDialog.value = true;
};

// 处理结算
const handleCheckout = () => {
  alert('正在生成财务账单结算数据...');
  // 这里可以添加实际的结算逻辑
};
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: var(--text-faint);
  border-radius: 10px;
}


</style>