import { post, put } from './request';

export interface Spec {
  name: string;
  pricePerKg: number;
  pricePerItem: number;
  total: number;
}

export interface ProSPecs {
  SID: number;
  Name: string;
  Specs: Spec[];
}

export interface CheckoutResponse {
  FieldList: string[];
  PBMap: Record<string, string[]>;
  Count: number;
  ProSPecs: ProSPecs[];
  TableData: string[];
  TotalRow: number[];
  PricePerKgRow: number[];
  PricePerItemRow: number[];
  TotalAmountRow?: number[];
  // 后端键为大写 TotalAmountRow（契约 §3.6）；旧小写别名 totalAmountRow 已随 T24 清理
}

export interface ProPriceRequest {
  productName: string;
  spec: string;
  total: number;
  pricePerKg: number;
  pricePerItem: number;
  totalAmount: number;
}

const toISO = (v: Date | string) => (v instanceof Date ? v.toISOString() : v);

export function getDayData(startTime: Date | string, endTime: Date | string) {
  return post<CheckoutResponse>('/api/checkout/daydata', {
    startTime: toISO(startTime),
    endTime: toISO(endTime),
  });
}

export function setProductBelong(mapp: Record<string, string[]>) {
  return put<null>('/api/checkout/product-belong', mapp);
}

export function setPrice(list: ProPriceRequest[]) {
  return post<CheckoutResponse>('/api/checkout/price', list);
}

export function exportCheckout() {
  return post<{ url: string }>('/api/checkout/export');
}
