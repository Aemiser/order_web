<template>
  <div class="table-card">
    <div class="table-wrapper">
      <table>
        <thead>
          <tr class="product-group">
            <th>日期 / 产品</th>
            <th colspan="2">香水柠檬</th>
            <th colspan="2">广香一级</th>
            <th colspan="2">广香二级</th>
            <th colspan="2">黄柠檬</th>
            <th colspan="2">金桔</th>
          </tr>
          <tr class="spec-header">
            <th></th>
            <th>5斤</th>
            <th>8斤</th>
            <th>10斤</th>
            <th>20斤</th>
            <th>7斤</th>
            <th>15斤</th>
            <th>3斤</th>
            <th>6斤</th>
            <th>2斤</th>
            <th>4斤</th>
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
            <td>447</td>
            <td>31</td>
            <td>168</td>
            <td>168</td>
            <td>1.6</td>
            <td>—</td>
            <td>34</td>
            <td>4.4</td>
            <td>168</td>
            <td>1.6</td>
          </tr>

          <!-- 单价行 -->
          <tr class="price-row">
            <td class="price-label">单价 (元/件)</td>
            <td>34</td>
            <td>40.8</td>
            <td>54.4</td>
            <td>16</td>
            <td>8</td>
            <td>—</td>
            <td>16</td>
            <td>8</td>
            <td>16</td>
            <td>8</td>
          </tr>

          <!-- 总金额行 -->
          <tr class="amount-row">
            <td class="amount-label">总金额 (元)</td>
            <td>15198</td>
            <td>122.4</td>
            <td>54.4</td>
            <td>25168</td>
            <td>1.6</td>
            <td>—</td>
            <td>544</td>
            <td>35.2</td>
            <td>2688</td>
            <td>12.8</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 底部说明 -->
    <div class="legend-section">
      <div class="legend-item">
        <span class="legend-color" style="background: rgba(59, 130, 246, 0.2);"></span>
        <span>香水柠檬: 5斤装(34元/件) · 8斤装(40.8元/件)</span>
      </div>
      <div class="legend-item">
        <span class="legend-color" style="background: rgba(20, 184, 166, 0.2);"></span>
        <span>广香一级: 10斤装(54.4元/件) · 20斤装(16元/件)</span>
      </div>
      <div class="legend-item">
        <span class="legend-color" style="background: rgba(139, 92, 246, 0.2);"></span>
        <span>金桔: 2斤装(16元/件) · 4斤装(8元/件)</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

// 表格数据
const tableData = ref([
  { date: '3月8日', values: [17, 4, 11, 12, '—', '—', 5, '—', 11, '—'] },
  { date: '3月9日', values: [11, 3, 8, 7, 2, '—', 4, 1, 8, 2] },
  { date: '3月10日', values: [30, 3, 11, 11, 2, '—', 4, '—', 12, '—'] },
  { date: '3月11日', values: [24, 7, 7, 11, 8, '—', 5, 1, 11, '—'] },
  { date: '3月12日', values: [17, 0, 9, 11, 2, '—', 3, '—', 12, 3] },
  { date: '3月13日', values: [7, 8, 5, 11, '—', 2, 4, '—', 8, '—'] },
  { date: '3月14日', values: [18, 1, 11, 11, '—', '—', 1, '—', 11, 1] },
  { date: '3月15日', values: [21, 1, 11, 11, '—', '—', 1, '—', 11, '—'] },
  { date: '3月16日', values: [23, 1, 11, 11, '—', '—', 1, '—', 11, '—'] },
  { date: '3月17日', values: [17, 1, 11, 11, '—', '—', 1, '—', 11, '—'] },
  { date: '3月18日', values: [20, 0, 3, 7, 1, 1, '—', '—', 11, '—'] }
])
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
  margin: 20px;
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