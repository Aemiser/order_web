import { del, get, post, put } from './request';

export interface ExcelAddressInfo {
  createdAt?: string;
  VID?: number;
  PreviewID?: number;
  收件人?: string;
  手机号?: string;
  收货地址?: string;
  商品规格?: string;
  备注?: string;
  // Go 端 JSON 用中文键；英文别名用于兼容既有组件的双写法读取
  Recipient?: string;
  Phone?: string;
  Address?: string;
  ProductSpecs?: string;
  Remarks?: string;
}

export interface OrderQueryResponse {
  list: ExcelAddressInfo[];
  count: number;
}

export interface RecognitionResult {
  createdAt?: string;
  VID?: number;
  PreviewID?: number;
  收件人?: string;
  手机号?: string;
  收货地址?: string;
  商品规格?: string;
  备注?: string;
  Recipient?: string;
  Phone?: string;
  Address?: string;
  ProductSpecs?: string;
  Remarks?: string;
  [key: string]: unknown;
}

const toISO = (v: Date | string) => (v instanceof Date ? v.toISOString() : v);

export function recognition(text: string, aimodel: number, previewType: number) {
  return post<RecognitionResult[]>('/api/order/recognition', { text, aimodel, previewType });
}

export function getPreview() {
  return get<ExcelAddressInfo[]>('/api/order/preview');
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
