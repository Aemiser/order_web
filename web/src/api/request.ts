import { getToken, notifyUnauthorized } from './auth';

export const BASE = import.meta.env.VITE_API_BASE ?? '';

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }
  // 携带登录令牌（JWT Bearer）；无令牌时后端受保护接口一律 401
  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  // 登录态失效：清空会话并踢回登录页（登录接口本身走业务码，不在此列）
  if (res.status === 401) {
    notifyUnauthorized();
    throw new Error('登录已过期，请重新登录');
  }

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }

  const payload = (await res.json()) as ApiResponse<T>;
  if (payload.code !== 0) {
    throw new Error(payload.msg || '请求失败');
  }
  return payload.data;
}

export const get = <T>(path: string) => request<T>('GET', path);
export const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body);
export const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body);
export const del = <T>(path: string) => request<T>('DELETE', path);
