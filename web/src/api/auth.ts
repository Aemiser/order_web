// auth.ts：登录态管理（token + 当前用户），localStorage 持久化。
// 全局唯一的 401 处理回调由 App.vue 注册，用于「踢回登录页」。
import { ref } from 'vue';

const TOKEN_KEY = 'order_web_token';
const USER_KEY = 'order_web_user';

export interface AuthUser {
  username: string;
  nickname: string;
  role: string;
}

// 模块级响应式登录态；App.vue 据此渲染登录页或主界面。
export const isAuthed = ref<boolean>(!!localStorage.getItem(TOKEN_KEY));
export const currentUser = ref<AuthUser | null>(readUser());

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setSession(token: string, user: AuthUser): void {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  isAuthed.value = true;
  currentUser.value = user;
}

export function clearSession(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  isAuthed.value = false;
  currentUser.value = null;
}

// 全局 401 处理器：App.vue 挂载时注册，请求层收到 401 时调用。
let unauthorizedHandler: (() => void) | null = null;

export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler;
}

export function notifyUnauthorized(): void {
  clearSession();
  if (unauthorizedHandler) {
    unauthorizedHandler();
  }
}

function readUser(): AuthUser | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    return null;
  }
}
