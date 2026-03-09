<template>
  <div class="table-card">
    <div class="table-wrapper">
      <table>
        <thead>
          <tr class="product-group">
            <th>日期 / 产品</th>
            <th v-for="product in products" :key="product.name" :colspan="product.specs.length">
              {{ product.name }}
            </th>
          </tr>
          <tr class="spec-header">
            <th></th>
            <template v-for="product in products" :key="product.name">
              <th v-for="spec in product.specs" :key="`${product.name}-${spec.name || spec}`">
                {{ spec.name || spec }}
              </th>
            </template>
          </tr>
        </thead>
        <tbody>
          <!-- 使用v-for动态渲染数据行 -->
          <tr v-for="(row, index) in tableData" :key="index" class="data-row">
            <td class="date-column">{{ row.date }}</td>
            <template v-for="(product, productIndex) in products" :key="product.name">
              <td v-for="(spec, specIndex) in product.specs" :key="`${productIndex}-${specIndex}`" class="spec-cell">
                <template v-if="row.values && row.values[getSpecIndex(productIndex, specIndex)] !== '—'">
                  <span class="spec-value">{{ row.values[getSpecIndex(productIndex, specIndex)] }}</span>
                </template>
                <template v-else>
                  <span class="empty-cell">—</span>
                </template>
              </td>
            </template>
          </tr>

          <!-- 总计行 -->
          <tr class="total-row">
            <td class="total-label">总计 (件)</td>
            <template v-for="(product, productIndex) in products" :key="product.name">
              <td v-for="(spec, specIndex) in product.specs" :key="`total-${productIndex}-${specIndex}`">
                {{ totalRow[getSpecIndex(productIndex, specIndex)] }}
              </td>
            </template>
          </tr>

          <!-- 元/斤行 -->
          <tr class="price-row">
            <td class="price-label">
              <template v-if="isEditingUnit">
                <div class="flex items-center gap-2">
                  <input 
                    type="text" 
                    v-model="editUnit"
                    @blur="saveUnit"
                    @keyup.enter="saveUnit"
                    @keyup.esc="cancelEditUnit"
                    class="bg-white/5 border border-white/10 rounded px-3 py-1 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors text-sm"
                    placeholder="输入计量单位"
                  />
                  <button 
                    @click="saveUnit"
                    class="p-1 rounded bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors"
                    title="保存"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                  </button>
                  <button 
                    @click="cancelEditUnit"
                    class="p-1 rounded bg-white/5 text-slate-400 hover:text-white transition-colors"
                    title="取消"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              </template>
              <template v-else>
                <span @click="startEditUnit" class="cursor-pointer hover:text-blue-400 transition-colors">
                  {{ unitOfMeasure }}
                </span>
              </template>
            </td>
            <template v-for="(product, productIndex) in products" :key="product.name">
              <td v-for="(spec, specIndex) in product.specs" :key="`kg-${productIndex}-${specIndex}`">
                <template v-if="isEditMode">
                  <input 
                    type="text" 
                    :value="editPricePerKg[getSpecIndex(productIndex, specIndex)] || ''"
                    @input="handlePricePerKgChange(getSpecIndex(productIndex, specIndex), ($event.target as HTMLInputElement).value)"
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors text-center"
                    placeholder="0.00"
                  />
                </template>
                <template v-else>
                  {{ pricePerKgRow[getSpecIndex(productIndex, specIndex)] }}
                </template>
              </td>
            </template>
          </tr>

          <!-- 元/件行 -->
          <tr class="price-row">
            <td class="price-label">元/件</td>
            <template v-for="(product, productIndex) in products" :key="product.name">
              <td v-for="(spec, specIndex) in product.specs" :key="`price-${productIndex}-${specIndex}`">
                <template v-if="isEditMode">
                  <input 
                    type="text" 
                    :value="editPrices[getSpecIndex(productIndex, specIndex)] || ''"
                    @input="handlePriceChange(getSpecIndex(productIndex, specIndex), ($event.target as HTMLInputElement).value)"
                    class="w-full bg-white/5 border border-white/10 rounded px-3 py-2 text-slate-200 focus:outline-none focus:border-blue-500 transition-colors text-center"
                    placeholder="0.00"
                  />
                </template>
                <template v-else>
                  {{ pricePerItemRow[getSpecIndex(productIndex, specIndex)] }}
                </template>
              </td>
            </template>
          </tr>

          <!-- 总金额行 -->
          <tr class="amount-row">
            <td class="amount-label">总金额 (元)</td>
            <template v-for="(product, productIndex) in products" :key="product.name">
              <td v-for="(spec, specIndex) in product.specs" :key="`amount-${productIndex}-${specIndex}`">
                {{ totalAmountRow[getSpecIndex(productIndex, specIndex)] }}
              </td>
            </template>
          </tr>
        </tbody>
      </table>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'

// 后端数据结构
interface ProSPecs {
  SID: number;
  Name: string;
  Specs: string[];
}

interface CheckoutResponse {
  FieldList: string[];
  Count: number;
  ProSPecs: ProSPecs[];
  TableData: string[];
  PBMap: Record<string, string[]>;
}

// 前端表格数据结构
interface TableRow {
  date: string;
  values: (number | string)[];
}

// 商品数据结构
interface Product {
  name: string;
  specs: string[];
}

// 商品列表
const products = ref<Product[]>([
  { name: '香水柠檬', specs: ['5斤', '8斤', '10斤', '20斤'] },
  { name: '广香一级', specs: ['10斤', '20斤'] },
  { name: '广香二级', specs: ['7斤', '15斤'] },
  { name: '黄柠檬', specs: ['3斤', '6斤'] },
  { name: '金桔', specs: ['2斤', '4斤'] }
]);

// 表格数据
const tableData = ref<TableRow[]>([])

// 单价数据
const prices = ref<(number | string)[]>([])

// 后端返回的汇总数据
const totalRow = ref<(number | string)[]>([])
const pricePerKgRow = ref<(number | string)[]>([])
const pricePerItemRow = ref<(number | string)[]>([])
const totalAmountRow = ref<(number | string)[]>([])

// 编辑模式状态
const isEditMode = ref(false)

// 本地编辑的价格数据
const editPrices = ref<(number | string)[]>([])
const editPricePerKg = ref<(number | string)[]>([])

// 计量单位状态
const unitOfMeasure = ref('元/斤')
const isEditingUnit = ref(false)
const editUnit = ref('元/斤')

// 模拟从后端获取数据
const fetchCheckoutData = async (): Promise<CheckoutResponse> => {
  // 实际项目中这里会调用API
  // const response = await fetch('/api/checkout');
  // return response.json();
  
  // 模拟API延迟
  return new Promise(resolve => {
    setTimeout(() => {
      resolve({
        FieldList: ['客户信息', '订单信息', '配送信息', '备注', '操作'],
        Count: 0,
        ProSPecs: [],
        TableData: [],
        PBMap: {
          '客户信息': ['客户姓名', '联系方式'],
          '配送信息': ['配送地址'],
          '订单要求': ['商品信息'],
          '备注说明': ['特殊要求'],
          '操作选项': ['打印选项']
        }
      });
    }, 500);
  });
};

// 数据转换函数：将后端数据转换为前端表格数据
const transformData = (data: CheckoutResponse): TableRow[] => {
  // 确保数据结构完整
  if (!data) {
    return [];
  }
  
  // 更新商品列表
  if (data.ProSPecs && data.ProSPecs.length > 0) {
    products.value = data.ProSPecs.map(proSpec => ({
      name: proSpec.Name || '',
      specs: (proSpec.Specs || []).map(spec => {
        // 如果 spec 是对象，确保它有 name 属性
        if (typeof spec === 'object' && spec !== null) {
          return { name: spec.name || spec.Name || JSON.stringify(spec) };
        }
        // 如果 spec 是字符串，直接使用
        return { name: spec };
      })
    }));
  } else {
    // 当没有商品数据时，清空商品列表
    products.value = [];
  }
  
  // 构建表格数据
  const tableRows: TableRow[] = [];
  
  // 解析 TableData
  if (data.TableData && data.TableData.length > 0) {
    // 假设 TableData 是按日期分组的字符串数组
    // 格式可能为: ["日期1,值1,值2,...", "日期2,值1,值2,..."]
    data.TableData.forEach(rowData => {
      if (typeof rowData === 'string') {
        const parts = rowData.split(',');
        if (parts.length > 0) {
          const date = parts[0] || '';
          const values = parts.slice(1).map(value => {
            // 尝试转换为数字，如果失败则保持字符串
            const numValue = parseFloat(value);
            return isNaN(numValue) ? value : numValue;
          });
          tableRows.push({ date, values });
        }
      }
    });
  }
  
  // 从后端数据中提取汇总数据
  totalRow.value = data.TotalRow || [];
  pricePerKgRow.value = data.PricePerKgRow || [];
  pricePerItemRow.value = data.PricePerItemRow || [];
  totalAmountRow.value = data.totalAmountRow || [];
  
  // 更新单价数据
  prices.value = pricePerItemRow.value;
  
  return tableRows;
};

// 处理测试数据生成
const handleGenerateTestData = (testData: CheckoutResponse) => {
  console.log('收到测试数据:', testData);
  
  const transformedData = transformData(testData);
  tableData.value = transformedData;
  console.log('测试数据已转换并更新表格');
};

// 定义组件事件
const emit = defineEmits<{
  (e: 'generate-test-data', testData: CheckoutResponse): void;
  (e: 'save-prices', prices: (number | string)[]): void;
  (e: 'save-unit', unit: string): void;
  (e: 'enter-edit-mode'): void;
  (e: 'exit-edit-mode'): void;
}>();

// 定义组件属性
const props = defineProps<{
  testData?: CheckoutResponse;
  checkoutResponse?: CheckoutResponse;
  isEditMode?: boolean;
}>();

// 监听编辑模式变化
watch(() => props.isEditMode, (newVal) => {
  if (newVal !== undefined) {
    if (!newVal && isEditMode.value) {
      // 退出编辑模式时，先保存价格数据
      savePrices();
    }
    
    isEditMode.value = newVal;
    if (newVal) {
      // 进入编辑模式时初始化编辑价格数据
      editPrices.value = [...pricePerItemRow.value];
      editPricePerKg.value = [...pricePerKgRow.value];
    } else {
      // 退出编辑模式时重置编辑价格数据
      editPrices.value = [];
      editPricePerKg.value = [];
    }
  }
}, { immediate: true });

// 监听后端响应数据变化
watch(() => props.checkoutResponse, (newVal) => {
  if (newVal) {
    console.log('收到后端响应数据:', newVal);
    // 更新表格数据
    const transformedData = transformData(newVal);
    tableData.value = transformedData;
    // 更新汇总数据
    totalRow.value = newVal.TotalRow || [];
    pricePerKgRow.value = newVal.PricePerKgRow || [];
    pricePerItemRow.value = newVal.PricePerItemRow || [];
    totalAmountRow.value = newVal.TotalAmountRow || [];
  }
}, { deep: true });

// 计算规格在一维数组中的索引
const getSpecIndex = (productIndex: number, specIndex: number): number => {
  let index = 0;
  for (let i = 0; i < productIndex; i++) {
    if (products.value[i] && products.value[i].specs) {
      index += products.value[i].specs.length;
    }
  }
  return index + specIndex;
};

// 进入编辑模式
const enterEditMode = () => {
  isEditMode.value = true;
  // 初始化编辑价格数据
  editPrices.value = [...pricePerItemRow.value];
  editPricePerKg.value = [...pricePerKgRow.value];
};

// 退出编辑模式
const exitEditMode = () => {
  isEditMode.value = false;
  // 重置编辑价格数据
  editPrices.value = [];
  editPricePerKg.value = [];
};

// 保存编辑的价格
const savePrices = () => {
  // 验证输入
  const isValid = editPrices.value.every(price => {
    if (price === '—' || price === '') return true;
    const numPrice = parseFloat(price as string);
    return !isNaN(numPrice) && numPrice >= 0;
  }) && editPricePerKg.value.every(price => {
    if (price === '—' || price === '') return true;
    const numPrice = parseFloat(price as string);
    return !isNaN(numPrice) && numPrice >= 0;
  });
  
  if (isValid) {
    // 构造 ProPriceRequest 数据结构
    const proPriceRequests = [];
    let specIndex = 0;
    
    products.value.forEach(product => {
      product.specs.forEach(spec => {
        const price = editPrices.value[specIndex];
        const pricePerKg = editPricePerKg.value[specIndex];
        if (price !== '—' && price !== '' || pricePerKg !== '—' && pricePerKg !== '') {
          const priceValue = price !== '—' && price !== '' ? parseFloat(price as string) : 0;
          const pricePerKgValue = pricePerKg !== '—' && pricePerKg !== '' ? parseFloat(pricePerKg as string) : 0;
          proPriceRequests.push({
            productName: product.name,
            spec: spec.name || spec,
            total: 0, // 后端会计算
            pricePerKg: pricePerKgValue,
            pricePerItem: priceValue,
            totalAmount: 0 // 后端会计算
          });
        }
        specIndex++;
      });
    });
    
    // 更新价格数据
    pricePerItemRow.value = [...editPrices.value];
    pricePerKgRow.value = [...editPricePerKg.value];
    // 退出编辑模式
    exitEditMode();
    // 触发保存事件，传递构造的数据
    emit('save-prices', proPriceRequests);
  } else {
    // 显示错误提示
    alert('请输入有效的价格');
  }
};

// 处理价格输入变化
const handlePriceChange = (index: number, value: string) => {
  // 只允许输入数字和小数点
  const numericValue = value.replace(/[^0-9.]/g, '');
  // 确保只有一个小数点
  const parts = numericValue.split('.');
  if (parts.length > 2) {
    return;
  }
  // 限制小数点后两位
  if (parts.length === 2 && parts[1].length > 2) {
    return;
  }
  editPrices.value[index] = numericValue;
};

// 处理 pricePerKg 输入变化
const handlePricePerKgChange = (index: number, value: string) => {
  // 只允许输入数字和小数点
  const numericValue = value.replace(/[^0-9.]/g, '');
  // 确保只有一个小数点
  const parts = numericValue.split('.');
  if (parts.length > 2) {
    return;
  }
  // 限制小数点后两位
  if (parts.length === 2 && parts[1].length > 2) {
    return;
  }
  editPricePerKg.value[index] = numericValue;
};

// 开始编辑计量单位
const startEditUnit = () => {
  isEditingUnit.value = true;
  editUnit.value = unitOfMeasure.value;
};

// 保存计量单位
const saveUnit = () => {
  // 验证输入
  if (editUnit.value.trim()) {
    unitOfMeasure.value = editUnit.value.trim();
    isEditingUnit.value = false;
    // 触发保存事件
    emit('save-unit', unitOfMeasure.value);
  } else {
    alert('请输入有效的计量单位');
  }
};

// 取消编辑计量单位
const cancelEditUnit = () => {
  isEditingUnit.value = false;
  editUnit.value = unitOfMeasure.value;
};

// 监听测试数据变化

// 监听 testData 变化
watch(() => props.testData, (newData) => {
  if (newData) {
    handleGenerateTestData(newData);
  }
}, { deep: true });

// 初始加载时处理数据
if (props.testData) {
  handleGenerateTestData(props.testData);
}

// 初始化数据
onMounted(async () => {
  try {
    const data = await fetchCheckoutData();
    console.log('从后端获取的结账数据:', data);
    
    // 转换数据并更新表格
    const transformedData = transformData(data);
    tableData.value = transformedData;
    
    console.log('转换后的表格数据:', transformedData);
  } catch (error) {
    console.error('获取结账数据失败:', error);
  }
});
</script>

<style scoped>
/* 字体导入 */
@import url('https://api.fontshare.com/v2/css?f[]=cabinet-grotesk@800&f[]=satoshi@400,500,700&display=swap');

/* 设计变量 */
:root {
  --bg-primary: #0F1115;
  --bg-card: #1E2228;
  --accent-blue: #3B82F6;
  --accent-teal: #14B8A6;
  --text-primary: #F1F5F9;
  --text-secondary: #94A3B8;
  --text-muted: #64748B;
  --border-light: rgba(255, 255, 255, 0.2);
  --border-lighter: rgba(255, 255, 255, 0.05);
  --row-hover: rgba(255, 255, 255, 0.02);
  --product-border: 1px solid rgba(59, 130, 246, 0.3);
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  background-color: var(--bg-primary);
  font-family: 'Satoshi', sans-serif;
  color: var(--text-primary);
}

/* 表格卡片 */
.table-card {
  background-color: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: 20px;
  padding: 24px 20px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  margin: 20px 0;
}

.table-wrapper {
  border-radius: 16px;
  border: 1px solid var(--border-lighter);
  overflow: auto;
  background-color: var(--bg-card);
}

table {
  width: 100%;
  border-collapse: collapse;
  min-width: 1000px;
  font-size: 0.95rem;
}

th {
  background: rgba(255, 255, 255, 0.02);
  color: var(--text-secondary);
  font-weight: 600;
  font-size: 0.95rem;
  padding: 18px 12px;
  white-space: nowrap;
  text-align: center;
}

.product-group th {
  background: rgba(59, 130, 246, 0.05);
  color: var(--accent-blue);
  font-size: 1rem;
  padding: 14px 8px;
}

.spec-header th {
  background: rgba(255, 255, 255, 0.01);
  color: var(--accent-teal);
}

td {
  padding: 14px 12px;
  text-align: center;
  vertical-align: middle;
  color: var(--text-primary);
}

.data-row:hover td {
  background-color: var(--row-hover);
}

.date-column {
  background: rgba(255, 255, 255, 0.02);
  font-weight: 600;
  color: var(--accent-teal);
  text-align: left;
  padding-left: 18px;
  font-size: 1rem;
  position: sticky;
  left: 0;
  background-color: var(--bg-card);
}

.spec-cell {
  font-weight: 500;
  color: var(--text-primary);
}

.spec-value {
  font-weight: 600;
  color: var(--accent-blue);
  font-size: 1.1rem;
}

.spec-unit {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-left: 2px;
  font-weight: 400;
}

.empty-cell {
  color: var(--text-muted);
  font-size: 0.9rem;
}

/* 总计行 */
.total-row {
  background: rgba(59, 130, 246, 0.05);
}

.total-row td {
  background: rgba(59, 130, 246, 0.05);
  font-weight: 600;
  color: var(--accent-blue);
}

.total-label {
  background: rgba(59, 130, 246, 0.1);
  font-weight: 700;
  color: var(--accent-blue);
  text-align: left;
  padding-left: 18px;
}

/* 单价行 */
.price-row {
  background: rgba(20, 184, 166, 0.03);
}

.price-row td {
  background: rgba(20, 184, 166, 0.03);
  color: var(--accent-teal);
}

.price-label {
  font-weight: 600;
  color: var(--accent-teal);
  text-align: left;
  padding-left: 18px;
}

/* 总金额行 */
.amount-row {
  background: rgba(139, 92, 246, 0.05);
}

.amount-row td {
  background: rgba(139, 92, 246, 0.05);
  font-weight: 600;
  color: #A78BFA;
}

.amount-label {
  font-weight: 700;
  color: #A78BFA;
  text-align: left;
  padding-left: 18px;
}



/* 底部图例 */
.legend-section {
  margin-top: 20px;
  padding: 0 12px;
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.legend-color {
  width: 10px;
  height: 10px;
  border-radius: 4px;
}

/* 响应式 */
@media (max-width: 768px) {
  .table-card {
    margin: 10px;
    padding: 16px 12px;
  }
}
</style>