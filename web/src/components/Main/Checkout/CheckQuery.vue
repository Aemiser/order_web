<template>
  <div class="bg-[#1E2228] border border-white/5 rounded-[2rem] p-6 mb-6">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-x-6 gap-y-4">
      <!-- 日期范围选择 -->
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
      
      <!-- 查询按钮 -->
      <div class="flex items-end space-x-3">
        <button 
          @click="handleQuery" 
          class="px-6 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-bold shadow-lg shadow-blue-600/20 transition-all flex items-center justify-center gap-2 active:scale-95 text-sm"
        >
          <Search class="w-4 h-4" />
          <span>立即查询</span>
        </button>
      </div>
      
      <!-- 错误提示 -->
      <div v-if="errors.dateRange" class="col-span-full mt-3">
        <p class="text-red-400 text-xs font-medium">{{ errors.dateRange }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { Search, Filter, Database, Calendar } from 'lucide-vue-next';
import { ElDatePicker } from 'element-plus';
import * as checkoutApi from '../../../api/checkout';
import type { CheckoutResponse } from '../../../api/checkout';

// 定义组件属性
const props = defineProps<{
  initialDateStart?: string;
  initialDateEnd?: string;
}>();

// 定义组件事件
const emit = defineEmits<{
  (e: 'query', params: {
    dateStart: string;
    dateEnd: string;
    category: string;
    keyword: string;
    data?: any;
  }): void;
  (e: 'show-field-dialog', data: {
    fieldList: string[];
    pbMap: Record<string, string[]>;
  }): void;
  (e: 'update-field-data', data: {
    fieldList: string[];
    pbMap: Record<string, string[]>;
  }): void;
  (e: 'field-data-loading', isLoading: boolean): void;
}>();

// 状态管理（ElDatePicker daterange 的 v-model 可能为 Date 对象对或日期字符串对）
const dateRange = ref<[Date, Date] | [string, string] | null>([
  props.initialDateStart || '2026-03-01',
  props.initialDateEnd || '2026-03-31'
]);
const selectedCategory = ref('所有品种');
const searchKeyword = ref('');

// 错误信息
const errors = ref({
  dateRange: ''
});

// 处理日期变化
const handleDateChange = () => {
  // 清除错误信息
  errors.value.dateRange = '';
};

// 处理查询
const handleQuery = async () => {
  try {
    // 验证日期输入
    if (!dateRange.value || dateRange.value.length !== 2) {
      errors.value.dateRange = '请选择日期范围';
      return;
    }
    
    // 验证日期格式
    if (!dateRange.value[0] || !dateRange.value[1]) {
      errors.value.dateRange = '请选择有效的日期范围';
      return;
    }
    
    // 处理日期对象或字符串
    let startDate, endDate;
    if (dateRange.value[0] instanceof Date) {
      // 如果是Date对象，直接使用
      startDate = new Date(dateRange.value[0]);
      endDate = new Date(dateRange.value[1]);
      // 设置时间
      startDate.setHours(0, 0, 0, 0);
      endDate.setHours(23, 59, 59, 999);
    } else {
      // 如果是字符串，创建UTC时间对象
      startDate = new Date(dateRange.value[0] + 'T00:00:00Z');
      endDate = new Date(dateRange.value[1] + 'T23:59:59Z');
    }
    
    // 验证日期对象是否有效
    if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
      errors.value.dateRange = '请选择有效的日期范围';
      return;
    }
    
    if (endDate < startDate) {
      errors.value.dateRange = '截至时间不能小于起始时间，请重新选择时间范围';
      return;
    }
    
    // 清除错误信息
    errors.value.dateRange = '';
    
    // 通知父组件数据正在加载
    emit('field-data-loading', true);

    // 调用后端日数据接口
    const response = await checkoutApi.getDayData(startDate, endDate);

    // 确保响应数据结构完整
    const validResponse: CheckoutResponse = {
      FieldList: response.FieldList || [],
      Count: response.Count || 0,
      ProSPecs: response.ProSPecs || [],
      TableData: response.TableData || [],
      PBMap: response.PBMap || {},
      TotalRow: response.TotalRow || [],
      PricePerKgRow: response.PricePerKgRow || [],
      PricePerItemRow: response.PricePerItemRow || [],
      TotalAmountRow: response.TotalAmountRow || [],
    };
    
    // 无论 FieldList 是否有值，都传递字段数据给父组件
    emit('update-field-data', {
      fieldList: validResponse.FieldList,
      pbMap: validResponse.PBMap
    });
    
    // 检查 FieldList 是否有值
    if (validResponse.FieldList && validResponse.FieldList.length > 0) {
      // 触发 FieldDialog 弹窗
      emit('show-field-dialog', {
        fieldList: validResponse.FieldList,
        pbMap: validResponse.PBMap
      });
    } else {
      // 发送查询事件，传递后端响应数据
      emit('query', {
        dateStart: String(dateRange.value[0]),
        dateEnd: String(dateRange.value[1]),
        category: selectedCategory.value,
        keyword: searchKeyword.value,
        data: validResponse
      });
    }
  } catch (error) {
    console.error('查询失败:', error);
    alert('查询失败，请稍后重试');
  } finally {
    // 通知父组件数据加载已完成
    emit('field-data-loading', false);
  }
};
</script>

<style scoped>
.glass-input {
  background-color: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
  color: white;
}

.glass-input:focus {
  outline: none;
  border-color: #3B82F6;
  background-color: rgba(255, 255, 255, 0.06);
}

.glass-input::placeholder {
  color: #94A3B8;
}

select.glass-input {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke-width='1.5' stroke='%2394A3B8' class='w-6 h-6'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' d='M19.5 8.25l-7.5 7.5-7.5-7.5' /%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.75rem center;
  background-size: 1.5em 1.5em;
  padding-right: 2.5rem;
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
</style>
