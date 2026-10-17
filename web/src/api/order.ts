import { del, get, post, put } from './request';

export interface ExcelAddressInfo {
  createdAt?: string;
  // DB 主键（后端键为小写 id；预览行恒为 0，历史订单行 >0，前端以 id>0 判定可删改）
  id?: number;
  ID?: number;
  VID?: number;
  PreviewID?: number;
  收件人?: string;
  手机号?: string;
  收货地址?: string;
  // 省份全称（34 省级行政区枚举，未识别为空串）
  省份?: string;
  商品规格?: string;
  备注?: string;
  // Go 端 JSON 用中文键；英文别名用于兼容既有组件的双写法读取
  Recipient?: string;
  Phone?: string;
  Address?: string;
  Province?: string;
  ProductSpecs?: string;
  Remarks?: string;
}

export interface OrderQueryResponse {
  // 后端空结果集为 nil slice，JSON 序列化为 null
  list: ExcelAddressInfo[] | null;
  count: number;
}

export interface RecognitionResult {
  createdAt?: string;
  id?: number;
  ID?: number;
  VID?: number;
  PreviewID?: number;
  收件人?: string;
  手机号?: string;
  收货地址?: string;
  省份?: string;
  商品规格?: string;
  备注?: string;
  Recipient?: string;
  Phone?: string;
  Address?: string;
  Province?: string;
  ProductSpecs?: string;
  Remarks?: string;
  [key: string]: unknown;
}

// 省份订单分布统计（GET /api/order/province-stats）
export interface ProvinceStat {
  name: string; // 省份全称（存储值，34 行政区枚举）
  shortName: string; // 省份简称（与 china.json 版本命名解耦，见契约 §3.1）
  value: number; // 订单数
}

export interface ProvinceStatsResponse {
  // 按 value 降序、同值按 name 升序；后端空结果集为 null
  list: ProvinceStat[] | null;
  total: number; // Σvalue（province 非空订单总数）
}

// 修改历史订单请求（PUT /api/order/:id）
// 中文键指针字段，全可选：未传 = 不更新；空串 = 清空（商品规格例外，不可清空只能替换）
export interface UpdateOrderRequest {
  收件人?: string;
  手机号?: string;
  收货地址?: string;
  省份?: string;
  商品规格?: string; // 字符串形态，格式【名称数字斤】，多项逗号分隔
  备注?: string;
}

// 预览审核结果（POST /api/order/preview/audit）
// 订单业务字段中文键（与 ExcelAddressInfo 一致）+ 英文别名双写；审核域字段英文小驼峰
export interface AuditItem {
  VID: number; // 预览行标识（与 ExcelAddressInfo.VID 一致；缺单项为 0）
  收件人: string;
  手机号: string;
  收货地址: string;
  商品规格: string; // 展示串
  Recipient?: string;
  Phone?: string;
  Address?: string;
  ProductSpecs?: string;
  diffType: 'missing' | 'wrong-key' | 'wrong-spec' | 'extra' | 'normal'; // 枚举值冻结，新增只能尾加
  diffDesc: string; // 程序对比差异描述（normal 为空串）
  aiVerdict: '真差异' | '误报' | ''; // 空串 = 未调 AI 或 AI 不可用（此时 aiFailed 为 true）
  aiSuggest: string; // AI 修正建议，无则空串
}

export interface AuditSummary {
  missing: number; // 缺单
  wrong: number; // 错单（wrong-key + wrong-spec 合计）
  extra: number; // 多单
  normal: number; // 正常
}

export interface AuditResponse {
  // 按 missing→wrong-key→wrong-spec→extra→normal 排序，同级按 VID 升序；后端空结果集为 null
  items: AuditItem[] | null;
  summary: AuditSummary;
  aiFailed: boolean; // AI 复核环节异常时 true（程序对比结果仍有效）
}

const toISO = (v: Date | string) => (v instanceof Date ? v.toISOString() : v);

export function recognition(text: string, aimodel: number, previewType: number) {
  return post<RecognitionResult[]>('/api/order/recognition', { text, aimodel, previewType });
}

export function getPreview() {
  return get<ExcelAddressInfo[] | null>('/api/order/preview');
}

export function setPreview(address: ExcelAddressInfo) {
  return put<null>('/api/order/preview', address);
}

export function deletePreview(id: number) {
  return del<null>(`/api/order/preview/${id}`);
}

export function createOrder(date: Date | string) {
  return post<null>('/api/order/create', { date: toISO(date) });
}

export function exportOrder() {
  return post<{ url: string }>('/api/order/export');
}

export function queryOrders(params: Record<string, unknown>) {
  return post<OrderQueryResponse>('/api/order/query', params);
}

// 省份订单分布统计。startTime/endTime 可选（RFC3339/ISO 字符串，须成对传入，都不传则全量）。
// request.ts 的 get 不支持 params，故在封装内自行拼接 encodeURIComponent query string。
export function getProvinceStats(startTime?: string, endTime?: string) {
  const params: string[] = [];
  if (startTime !== undefined) {
    params.push(`startTime=${encodeURIComponent(startTime)}`);
  }
  if (endTime !== undefined) {
    params.push(`endTime=${encodeURIComponent(endTime)}`);
  }
  const query = params.length > 0 ? `?${params.join('&')}` : '';
  return get<ProvinceStatsResponse>(`/api/order/province-stats${query}`);
}

// 修改历史订单（软删行/不存在 id 后端返回 50000；只传需要修改的字段）
export function updateOrder(id: number, data: UpdateOrderRequest) {
  return put<null>(`/api/order/${id}`, data);
}

// 软删除历史订单（非幂等：重复删除后端返回 50000 订单不存在，调用方成功后应移除该行）
export function deleteOrder(id: number) {
  return del<null>(`/api/order/${id}`);
}

// 预览审核（原始文本由后端 LastInputText 暂存，请求体为空 JSON 对象）
export function auditPreview() {
  return post<AuditResponse>('/api/order/preview/audit', {});
}
