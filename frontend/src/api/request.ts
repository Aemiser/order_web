export const BASE = import.meta.env.VITE_API_BASE ?? '';

export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

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
