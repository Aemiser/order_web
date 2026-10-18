<script setup lang="ts">
// 省份订单分布地图（T18）
// 数据源：GET /api/order/province-stats，走 api/order.ts 封装（F3）。
// 地图数据：本地 GeoJSON（DataV.GeoAtlas areas_v3 全称版，含九段线/南海诸岛要素），
// 构建期内联进产物，禁止运行时从第三方 CDN 拉取（F7）。
import { ref, onMounted, onUnmounted, watch } from 'vue';
import { ElMessage } from 'element-plus';
import * as echarts from 'echarts/core';
import { MapChart } from 'echarts/charts';
import { TooltipComponent, VisualMapComponent, TitleComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import chinaJsonRaw from '../../../assets/china.json?raw';
import * as orderApi from '../../../api/order';
import type { ProvinceStat } from '../../../api/order';

// ---- 本地 GeoJSON 最小类型（type alias 以获得隐式 index signature，可赋给 echarts 的 GeoJSON 入参）----
// geometry 的 type 与 coordinates 必须配对联合（与 echarts GeoJSONGeometry 结构一致）
type ChinaGeoGeometry =
  | { type: 'Polygon'; coordinates: number[][][] }
  | { type: 'MultiPolygon'; coordinates: number[][][][] };
type ChinaGeoFeature = {
  type: 'Feature';
  properties: { name?: string };
  geometry: ChinaGeoGeometry;
};
type ChinaGeoJSON = {
  type: 'FeatureCollection';
  features: ChinaGeoFeature[];
};

echarts.use([MapChart, TooltipComponent, VisualMapComponent, TitleComponent, CanvasRenderer]);

// 注册本地中国地图：全称版（"福建省"等 34 省级行政区）+ 九段线要素（adcode 100000_JD，name 为空串）
const chinaGeo = JSON.parse(chinaJsonRaw) as ChinaGeoJSON;
echarts.registerMap('china', chinaGeo);

// 地图区域名集合：用于后端 name/shortName → 地图区域名的映射，兼容全称版/简称版两种 GeoJSON
const regionNames = new Set(
  chinaGeo.features
    .map((feature) => feature.properties.name)
    .filter((name): name is string => typeof name === 'string' && name !== '')
);

const props = defineProps<{ startTime?: string; endTime?: string }>();

const chartRef = ref<HTMLDivElement>();
let chart: echarts.EChartsType | null = null;

// 后端统计项 → 地图数据项。
// 匹配策略（契约 Q1）：优先 shortName（简称版 GeoJSON 命名），回退 name 全称（本项目所用全称版命名）；
// 两者均不匹配时登记未匹配省名，便于联调排查（该省在地图上不着色）。
const toMapData = (list: ProvinceStat[]): Array<{ name: string; value: number }> => {
  const data: Array<{ name: string; value: number }> = [];
  const unmatched: string[] = [];
  for (const stat of list) {
    let regionName: string | null = null;
    if (stat.shortName !== '' && regionNames.has(stat.shortName)) {
      regionName = stat.shortName;
    } else if (stat.name !== '' && regionNames.has(stat.name)) {
      regionName = stat.name;
    }
    if (regionName === null) {
      unmatched.push(stat.name);
      continue;
    }
    data.push({ name: regionName, value: stat.value });
  }
  if (unmatched.length > 0) {
    console.warn('[ProvinceMap] 省份统计项与地图区域名不匹配：', unmatched.join('、'));
  }
  return data;
};

const renderChart = (list: ProvinceStat[] | null) => {
  if (chart === null) return;
  const data = toMapData(list ?? []);
  // 空数据/低峰值时保底 10，避免 visualMap 色带退化
  const maxValue = Math.max(10, ...data.map((item) => item.value));
  chart.setOption({
    tooltip: {
      trigger: 'item',
      // 无数据省份 value 为 NaN，显示为 0
      formatter: (params: { name?: string; value?: unknown }) => {
        const value =
          typeof params.value === 'number' && !Number.isNaN(params.value) ? params.value : 0;
        return `${params.name ?? ''}: ${value} 单`;
      },
    },
    visualMap: {
      min: 0,
      max: maxValue,
      left: 'right',
      top: 'bottom',
      text: ['高', '低'],
      inRange: { color: ['#e0f2fe', '#0284c7'] },
      calculable: true,
    },
    series: [
      {
        name: '订单数',
        type: 'map',
        map: 'china',
        roam: true,
        data,
        label: { show: true, fontSize: 9, color: 'var(--text-muted)' },
        itemStyle: { areaColor: 'var(--bg-card)', borderColor: 'var(--border-strong)' },
        emphasis: { label: { show: true }, itemStyle: { areaColor: 'var(--accent)' } },
      },
    ],
  });
};

const loadData = async () => {
  try {
    const resp = await orderApi.getProvinceStats(props.startTime, props.endTime);
    renderChart(resp.list);
  } catch (error) {
    console.error('[ProvinceMap] 省份统计数据加载失败:', error);
    ElMessage.error('省份统计数据加载失败');
    renderChart(null);
  }
};

onMounted(() => {
  if (chartRef.value !== undefined) {
    chart = echarts.init(chartRef.value);
    loadData();
  }
});

onUnmounted(() => {
  chart?.dispose();
  chart = null;
});

// 查询时间范围变化时联动刷新（详情页查询条件联动，架构 §4.6）
watch([() => props.startTime, () => props.endTime], () => {
  loadData();
});
</script>

<template>
  <div ref="chartRef" class="w-full h-[480px]"></div>
</template>
