import { post, get } from './request';
import type { AuthUser } from './auth';

export interface LoginResult {
  token: string;
  expiresIn: number;
  user: AuthUser;
}

// POST /api/auth/login（匿名接口，返回业务码而非 HTTP 401）
export function login(username: string, password: string) {
  return post<LoginResult>('/api/auth/login', { username, password });
}

// GET /api/auth/me：获取当前账号最新信息（昵称、角色）
export function me() {
  return get<AuthUser>('/api/auth/me');
}

// POST /api/auth/logout：服务端审计留痕；前端负责丢弃令牌
export function logout() {
  return post<null>('/api/auth/logout');
}
