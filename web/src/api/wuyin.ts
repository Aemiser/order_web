import { getToken } from './auth';
import { BASE, get, post, put, del } from './request';

// 吴饮（吾饮良品）品牌链路 API。与 api/order.ts（一点点）平行：
// 字段多 标签/发送人，商品规格为件计价字符串（【香水柠檬1件】）。

export interface WuyinAddressInfo {
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
  标签?: string;
  发送人?: string;
  // Go 端英文别名双写（与 ExcelAddressInfo 同款读取策略）
  Recipient?: string;
  Phone?: string;
  Address?: string;
  Province?: string;
  ProductSpecs?: string;
  Remarks?: string;
  Tag?: string;
  Sender?: string;
}

export interface WuyinOrderQueryResponse {
  // 后端空结果集为 nil slice，JSON 序列化为 null
  list: WuyinAddressInfo[] | null;
  count: number;
}

export interface WuyinTagConfig {
  ID?: number;
  id?: number;
  name: string;
  keywords: string[] | null;
  sort?: number;
}

export interface WuyinTagStat {
  tag: string;
  count: number;
}

export interface WuyinOrderQueryParams {
  page?: number;
  limit?: number;
  key?: string;
  tags?: string[];
  addressInfo?: {
    标签?: string;
    收件人?: string;
    手机号?: string;
  };
  startTime?: string;
  endTime?: string;
}

const toISO = (v: Date | string) => (v instanceof Date ? v.toISOString() : v);

export function recognition(text: string, aimodel: number, previewType: number) {
  return post<WuyinAddressInfo[] | null>('/api/wuyin/recognition', { text, aimodel, previewType });
}

export function getPreview() {
  return get<WuyinAddressInfo[] | null>('/api/wuyin/preview');
}

export function setPreview(address: WuyinAddressInfo) {
  return put<null>('/api/wuyin/preview', address);
}

export function deletePreview(vid: number) {
  return del<null>(`/api/wuyin/preview/${vid}`);
}

export function createOrder(date: Date | string) {
  return post<null>('/api/wuyin/create', { date: toISO(date) });
}

export function exportOrder() {
  return post<{ url: string }>('/api/wuyin/export');
}

export function queryOrders(params: WuyinOrderQueryParams) {
  return post<WuyinOrderQueryResponse>('/api/wuyin/query', params);
}

export function updateOrder(id: number, data: Partial<Record<string, string>>) {
  return put<null>(`/api/wuyin/${id}`, data);
}

export function deleteOrder(id: number) {
  return del<null>(`/api/wuyin/${id}`);
}

// 标签订单数统计（startTime/endTime 可选且成对，RFC3339）
export function getTagStats(startTime?: string, endTime?: string) {
  const params: string[] = [];
  if (startTime !== undefined) params.push(`startTime=${encodeURIComponent(startTime)}`);
  if (endTime !== undefined) params.push(`endTime=${encodeURIComponent(endTime)}`);
  const query = params.length > 0 ? `?${params.join('&')}` : '';
  return get<WuyinTagStat[] | null>(`/api/wuyin/tag-stats${query}`);
}

export function getTags() {
  return get<WuyinTagConfig[] | null>('/api/wuyin/tags');
}

export function createTag(data: { name: string; keywords: string[]; sort?: number }) {
  return post<null>('/api/wuyin/tags', data);
}

export function updateTag(id: number, data: { name: string; keywords: string[]; sort?: number }) {
  return put<null>(`/api/wuyin/tags/${id}`, data);
}

export function deleteTag(id: number) {
  return del<null>(`/api/wuyin/tags/${id}`);
}

// importExcel 表格导入（multipart）。request.ts 无表单封装，且不该为一品牌扩展公共层，
// 故在本文件内自持：鉴权头照抄 request.ts 约定。
export async function importExcel(file: File, tag: string): Promise<WuyinAddressInfo[] | null> {
  const form = new FormData();
  form.append('file', file);
  if (tag) form.append('tag', tag);

  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${BASE}/api/wuyin/import`, { method: 'POST', headers, body: form });
  if (res.status === 401) throw new Error('登录已过期，请重新登录');
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const payload = (await res.json()) as { code: number; msg: string; data: WuyinAddressInfo[] | null };
  if (payload.code !== 0) throw new Error(payload.msg || '导入失败');
  return payload.data;
}

// streamRecognition SSE 流式识别（解析逻辑与 api/stream.ts 同构，端点换吴饮；
// 不改 stream.ts 是为保持一点点路径零改动）
export async function streamRecognition(
  text: string,
  aimodel: number,
  previewType: number,
  handlers: { onData: (data: string) => void; onDone?: () => void },
): Promise<void> {
  const token = getToken();
  const res = await fetch(`${BASE}/api/wuyin/recognition/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ text, aimodel, previewType }),
  });

  if (!res.ok || !res.body) throw new Error(`HTTP ${res.status}`);
  const contentType = res.headers.get('content-type') ?? '';
  if (!contentType.includes('text/event-stream')) {
    const payload = await res.json().catch(() => null);
    throw new Error(payload?.msg || `HTTP ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    let idx: number;
    while ((idx = buffer.indexOf('\n\n')) >= 0) {
      const raw = buffer.slice(0, idx);
      buffer = buffer.slice(idx + 2);
      const data = raw
        .split('\n')
        .filter(line => line.startsWith('data: '))
        .map(line => line.slice(6))
        .join('\n');
      if (data === 'DONE') {
        handlers.onDone?.();
        return;
      }
      if (data) handlers.onData(data);
    }
  }
  throw new Error('SSE 流未正常结束（未收到 DONE）');
}

// ---- 结账（件计价 × 标签分组）----

export interface WuyinCheckoutProduct {
  name: string;
  spec: string; // 规格串，如 "9斤"
  itemsPerDate: number[] | null;
  totalItems: number; // 件数（= 该规格条目数）
  pricePerItem: number;
  totalAmount: number;
}

export interface WuyinCheckoutGroup {
  tag: string;
  dates: string[] | null;
  products: WuyinCheckoutProduct[] | null;
  count: number;
  grandTotal: number;
}

// 按标签分组的结账数据（tags 可选，空 = 全部标签）
export function getDayData(startTime: string, endTime: string, tags?: string[]) {
  return post<WuyinCheckoutGroup[] | null>('/api/wuyin/checkout/daydata', { startTime, endTime, tags });
}

// 保存单价（元/件）并返回重算后的分组账单；价格按 商品+规格 唯一
export function setPrice(list: { productName: string; spec: string; pricePerItem: number }[]) {
  return post<WuyinCheckoutGroup[] | null>('/api/wuyin/checkout/price', list);
}

// 结账导出（tags 可选，空 = 全部标签；每个标签一个 sheet）
export function exportCheckout(tags?: string[]) {
  return post<{ url: string }>('/api/wuyin/checkout/export', { tags });
}
