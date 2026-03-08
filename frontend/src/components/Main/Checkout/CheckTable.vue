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
              <th v-for="spec in product.specs" :key="`${product.name}-${spec}`">
                {{ spec }}
              </th>
            </template>
          </tr>
        </thead>
        <tbody>
          <!-- 使用v-for动态渲染数据行 -->
          <tr v-for="(row, index) in tableData" :key="index" class="data-row">
            <td class="date-column">{{ row.date }}</td>
            <td v-for="(value, idx) in row.values" :key="idx" class="spec-cell">
              <template v-if="value !== '—'">
                <span class="spec-value">{{ value }}</span>
                <span class="spec-unit">件</span>
              </template>
              <template v-else>
                <span class="empty-cell">—</span>
              </template>
            </td>
          </tr>

          <!-- 总计行 -->
          <tr class="total-row">
            <td class="total-label">总计 (件)</td>
            <td v-for="(value, index) in totalRow" :key="`total-${index}`">
              {{ value }}
            </td>
          </tr>

          <!-- 元/斤行 -->
          <tr class="price-row">
            <td class="price-label">元/斤</td>
            <td v-for="(value, index) in pricePerKgRow" :key="`kg-${index}`">
              {{ value }}
            </td>
          </tr>

          <!-- 单价行 -->
          <tr class="price-row">
            <td class="price-label">元/件</td>
            <td v-for="(value, index) in prices" :key="`price-${index}`">
              {{ value }}
            </td>
          </tr>

          <!-- 总金额行 -->
          <tr class="amount-row">
            <td class="amount-label">总金额 (元)</td>
            <td v-for="(value, index) in amountRow" :key="`amount-${index}`">
              {{ value }}
            </td>
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

// 计算属性：总计行数据
const totalRow = computed(() => {
  if (tableData.value.length === 0) {
    return []
  }
  
  const totalValues: (number | string)[] = []
  const firstRow = tableData.value[0]
  
  if (firstRow && firstRow.values) {
    for (let i = 0; i < firstRow.values.length; i++) {
      let sum = 0
      let hasValue = false
      
      for (const row of tableData.value) {
        if (row.values && row.values[i] !== '—' && row.values[i] !== '' && row.values[i] !== null && row.values[i] !== undefined) {
          const value = Number(row.values[i])
          if (!isNaN(value)) {
            sum += value
            hasValue = true
          }
        }
      }
      
      totalValues.push(hasValue ? sum : '—')
    }
  }
  
  return totalValues
})

// 计算属性：元/斤行数据
const pricePerKgRow = computed(() => {
  if (prices.value.length === 0) {
    return []
  }
  
  return prices.value.map(price => {
    if (price === '—' || price === '' || price === null || price === undefined) {
      return '—'
    }
    const priceValue = Number(price)
    return !isNaN(priceValue) ? priceValue / 2 : '—'
  })
})

// 计算属性：总金额行数据
const amountRow = computed(() => {
  if (totalRow.value.length === 0 || prices.value.length === 0) {
    return []
  }
  
  return totalRow.value.map((total, index) => {
    if (total === '—' || total === '' || total === null || total === undefined) {
      return '—'
    }
    if (index >= prices.value.length) {
      return '—'
    }
    const price = prices.value[index]
    if (price === '—' || price === '' || price === null || price === undefined) {
      return '—'
    }
    const totalValue = Number(total)
    const priceValue = Number(price)
    return !isNaN(totalValue) && !isNaN(priceValue) ? totalValue * priceValue : '—'
  })
})

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
      specs: proSpec.Specs || []
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
  
  // 模拟单价数据（实际项目中应该从后端获取）
  // 这里根据商品规格数量生成对应的单价数据
  const totalSpecsCount = products.value.reduce((total, product) => total + (product.specs ? product.specs.length : 0), 0);
  const newPrices: (number | string)[] = [];
  for (let i = 0; i < totalSpecsCount; i++) {
    // 模拟不同规格的单价
    const basePrice = 10 + (i * 2);
    newPrices.push(basePrice);
  }
  prices.value = newPrices;
  
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
}>();

// 定义组件属性
const props = defineProps<{
  testData?: CheckoutResponse;
}>();

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
  --border-light: rgba(255, 255, 255, 0.1);
  --border-lighter: rgba(255, 255, 255, 0.05);
  --row-hover: rgba(255, 255, 255, 0.02);
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
  border-bottom: 2px solid var(--border-light);
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
  border-bottom: 1px solid var(--border-lighter);
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
  border-right: 1px solid var(--border-light);
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
  border-top: 2px solid var(--border-light);
  border-bottom: 2px solid var(--border-light);
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
  border-bottom: 1px dashed var(--border-light);
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
  border-top: 2px solid var(--border-light);
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