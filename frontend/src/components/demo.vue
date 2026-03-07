<template>
  <div class="multi-dim-table-demo">
    <h1>Vue 多维表格示例</h1>

    <!-- 示例1: 多维表头（合并单元格） -->
    <section class="demo-section">
      <h2>📊 示例1: 多维表头（季度/月度销售报表）</h2>
      <div class="table-container">
        <table class="multi-header-table">
          <!-- 第一层表头：产品 + 各季度 -->
          <thead>
          <tr>
            <th :rowspan="2" class="product-col">产品名称</th>
            <th :colspan="3" class="quarter">2024年第一季度</th>
            <th :colspan="3" class="quarter">2024年第二季度</th>
            <th :rowspan="2" class="total-col">总计</th>
          </tr>
          <!-- 第二层表头：各月份 -->
          <tr>
            <th>1月</th>
            <th>2月</th>
            <th>3月</th>
            <th>4月</th>
            <th>5月</th>
            <th>6月</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="(item, index) in salesData" :key="index">
            <td class="product-name">{{ item.product }}</td>
            <td>{{ item.jan }}</td>
            <td>{{ item.feb }}</td>
            <td>{{ item.mar }}</td>
            <td>{{ item.apr }}</td>
            <td>{{ item.may }}</td>
            <td>{{ item.jun }}</td>
            <td class="total">{{ item.total }}</td>
          </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- 示例2: 数据透视效果（多维分组汇总） -->
    <section class="demo-section">
      <h2>📈 示例2: 多维数据透视（按区域+产品类别汇总）</h2>

      <!-- 控制面板 -->
      <div class="control-panel">
        <div class="filter-group">
          <label>行维度：</label>
          <select v-model="rowDimension">
            <option value="region">销售区域</option>
            <option value="category">产品类别</option>
          </select>
        </div>
        <div class="filter-group">
          <label>列维度：</label>
          <select v-model="colDimension">
            <option value="category">产品类别</option>
            <option value="region">销售区域</option>
          </select>
        </div>
        <div class="filter-group">
          <label>汇总方式：</label>
          <select v-model="aggregateType">
            <option value="sum">求和</option>
            <option value="avg">平均值</option>
            <option value="count">计数</option>
          </select>
        </div>
      </div>

      <!-- 透视表展示 -->
      <div class="table-container" v-if="pivotData.length">
        <table class="pivot-table">
          <thead>
          <tr>
            <th>{{ rowDimension === 'region' ? '销售区域' : '产品类别' }}</th>
            <th v-for="col in pivotColumns" :key="col">
              {{ col }}
            </th>
            <th>总计</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="row in pivotData" :key="row.name">
            <td class="row-header">{{ row.name }}</td>
            <td v-for="col in pivotColumns" :key="col"
                :class="{ 'highlight': row[col] > 10000 }">
              {{ formatValue(row[col]) }}
            </td>
            <td class="total">{{ formatValue(row.total) }}</td>
          </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- 示例3: 树形表格（多层次数据） -->
    <section class="demo-section">
      <h2>🌲 示例3: 树形多维表格（产品分类层级）</h2>
      <div class="table-container">
        <table class="tree-table">
          <thead>
          <tr>
            <th>产品分类</th>
            <th>一季度销量</th>
            <th>二季度销量</th>
            <th>同比增长</th>
          </tr>
          </thead>
          <tbody>
          <template v-for="item in treeData" :key="item.name">
            <!-- 一级分类 -->
            <tr class="level-0">
              <td>
                  <span class="expand-icon" @click="toggleExpand(item)">
                    {{ item.expanded ? '▼' : '▶' }}
                  </span>
                {{ item.name }}
              </td>
              <td>{{ item.q1 }}</td>
              <td>{{ item.q2 }}</td>
              <td :class="{ 'positive': item.growth > 0, 'negative': item.growth < 0 }">
                {{ item.growth }}%
              </td>
            </tr>
            <!-- 二级分类（子项） -->
            <tr v-for="child in item.children"
                v-show="item.expanded"
                :key="child.name"
                class="level-1">
              <td>└─ {{ child.name }}</td>
              <td>{{ child.q1 }}</td>
              <td>{{ child.q2 }}</td>
              <td :class="{ 'positive': child.growth > 0, 'negative': child.growth < 0 }">
                {{ child.growth }}%
              </td>
            </tr>
          </template>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

// 示例1: 销售数据
const salesData = ref([
  { product: '笔记本电脑', jan: 120, feb: 135, mar: 150, apr: 165, may: 180, jun: 210, total: 960 },
  { product: '智能手机', jan: 230, feb: 245, mar: 260, apr: 275, may: 290, jun: 320, total: 1620 },
  { product: '平板电脑', jan: 80, feb: 95, mar: 110, apr: 125, may: 140, jun: 155, total: 705 },
  { product: '智能手表', jan: 45, feb: 55, mar: 65, apr: 75, may: 85, jun: 95, total: 420 }
])

// 示例2: 原始数据（用于透视）
const sourceData = ref([
  { region: '华北', category: '电子产品', sales: 12500 },
  { region: '华北', category: '电子产品', sales: 8300 },
  { region: '华北', category: '服装', sales: 4200 },
  { region: '华北', category: '服装', sales: 3800 },
  { region: '华东', category: '电子产品', sales: 15800 },
  { region: '华东', category: '电子产品', sales: 9200 },
  { region: '华东', category: '服装', sales: 5600 },
  { region: '华东', category: '服装', sales: 4900 },
  { region: '华南', category: '电子产品', sales: 11200 },
  { region: '华南', category: '电子产品', sales: 7800 },
  { region: '华南', category: '服装', sales: 6300 },
  { region: '华南', category: '服装', sales: 5200 }
])

// 透视表配置
const rowDimension = ref('region')
const colDimension = ref('category')
const aggregateType = ref('sum')

// 计算透视列（去重后的类别）
const pivotColumns = computed(() => {
  const values = sourceData.value.map(item => item[colDimension.value])
  return [...new Set(values)]
})

// 计算透视行（去重后的行维度值）
const pivotRows = computed(() => {
  const values = sourceData.value.map(item => item[rowDimension.value])
  return [...new Set(values)]
})

// 计算透视数据
const pivotData = computed(() => {
  return pivotRows.value.map(rowName => {
    const rowData = { name: rowName, total: 0 }

    // 计算每个列的值
    pivotColumns.value.forEach(colName => {
      const filtered = sourceData.value.filter(item =>
          item[rowDimension.value] === rowName &&
          item[colDimension.value] === colName
      )

      let value = 0
      if (aggregateType.value === 'sum') {
        value = filtered.reduce((sum, item) => sum + item.sales, 0)
      } else if (aggregateType.value === 'avg') {
        value = filtered.length ? filtered.reduce((sum, item) => sum + item.sales, 0) / filtered.length : 0
      } else if (aggregateType.value === 'count') {
        value = filtered.length
      }

      rowData[colName] = value
      rowData.total += value
    })

    return rowData
  })
})

// 格式化数字
const formatValue = (value) => {
  if (typeof value !== 'number') return value
  return new Intl.NumberFormat('zh-CN').format(Math.round(value * 100) / 100)
}

// 示例3: 树形数据
const treeData = ref([
  {
    name: '消费电子产品',
    q1: 1250,
    q2: 1480,
    growth: 18.4,
    expanded: true,
    children: [
      { name: '智能手机', q1: 580, q2: 720, growth: 24.1 },
      { name: '笔记本电脑', q1: 420, q2: 460, growth: 9.5 },
      { name: '平板电脑', q1: 250, q2: 300, growth: 20.0 }
    ]
  },
  {
    name: '家用电器',
    q1: 890,
    q2: 950,
    growth: 6.7,
    expanded: false,
    children: [
      { name: '冰箱', q1: 320, q2: 340, growth: 6.3 },
      { name: '洗衣机', q1: 280, q2: 310, growth: 10.7 },
      { name: '空调', q1: 290, q2: 300, growth: 3.4 }
    ]
  },
  {
    name: '服装鞋帽',
    q1: 670,
    q2: 820,
    growth: 22.4,
    expanded: false,
    children: [
      { name: '男装', q1: 280, q2: 350, growth: 25.0 },
      { name: '女装', q1: 390, q2: 470, growth: 20.5 }
    ]
  }
])

// 展开/折叠
const toggleExpand = (item) => {
  item.expanded = !item.expanded
}
</script>

<style scoped>
.multi-dim-table-demo {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  font-family: 'Microsoft YaHei', sans-serif;
}

.demo-section {
  margin-bottom: 40px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

h1 {
  color: #2c3e50;
  text-align: center;
  margin-bottom: 30px;
}

h2 {
  font-size: 1.2rem;
  color: #34495e;
  margin-bottom: 15px;
  border-left: 4px solid #3498db;
  padding-left: 10px;
}

.table-container {
  overflow-x: auto;
  background: white;
  border-radius: 6px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.05);
}

table {
  width: 100%;
  border-collapse: collapse;
  min-width: 600px;
}

th, td {
  border: 1px solid #e0e0e0;
  padding: 12px 8px;
  text-align: center;
  font-size: 14px;
}

th {
  background: #f1f3f4;
  font-weight: 600;
  color: #2c3e50;
}

.multi-header-table th {
  background: #e8eaf6;
}

.quarter {
  background: #c5cae9;
  color: #1a237e;
}

.product-col, .product-name {
  background: #f5f5f5;
  font-weight: 500;
}

.total-col, .total {
  background: #ffecb3;
  font-weight: 600;
  color: #bf360c;
}

/* 控制面板样式 */
.control-panel {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
  padding: 15px;
  background: white;
  border-radius: 6px;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-group label {
  font-weight: 500;
  color: #555;
}

.filter-group select {
  padding: 6px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: white;
  cursor: pointer;
}

.filter-group select:hover {
  border-color: #3498db;
}

/* 透视表样式 */
.pivot-table th {
  background: #e0f2f1;
  color: #00695c;
}

.pivot-table .row-header {
  background: #f5f5f5;
  font-weight: 500;
  text-align: left;
  padding-left: 15px;
}

.pivot-table td.highlight {
  background: #fff9c4;
  font-weight: 600;
  color: #f57c00;
}

/* 树形表格样式 */
.tree-table .level-0 {
  background: #f3e5f5;
  font-weight: 600;
}

.tree-table .level-1 {
  background: #fafafa;
  color: #555;
}

.tree-table .level-1 td:first-child {
  padding-left: 35px;
}

.expand-icon {
  display: inline-block;
  width: 20px;
  cursor: pointer;
  color: #7b1fa2;
  user-select: none;
  margin-right: 5px;
}

.positive {
  color: #27ae60;
  font-weight: 600;
}

.negative {
  color: #c0392b;
  font-weight: 600;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .demo-section {
    padding: 15px;
  }

  th, td {
    padding: 8px 4px;
    font-size: 12px;
  }

  .control-panel {
    flex-direction: column;
    gap: 10px;
  }
}
</style>