import { http } from '@/lib/http';

export type AuthConfig = {
  oidcEnabled: boolean;
  oidcReady: boolean;
  oidcError?: string;
  providerName: string;
  allowLocalLogin: boolean;
};

export type AuthSession = {
  authenticated: boolean;
  username: string;
  displayName?: string;
  email?: string;
  authSource: string;
  admin: boolean;
};

type AuthResponse<T> = {
  code: number;
  msg?: string;
  data: T;
};

export function login(username: string, password: string) {
  const data = {
    username,
    password
  };
  return http.post('/api/auth/login', data);
}

export function logout() {
  return http.post('/api/auth/logout');
}

export function getConfig(): Promise<AuthResponse<AuthConfig>> {
  return http.get('/api/auth/config');
}

export function getSession(): Promise<AuthResponse<AuthSession>> {
  return http.get('/api/auth/session');
}

export function getAccount() {
  return http.get('/api/auth/account');
}

export function changePassword(username: string, password: string) {
  const data = {
    username,
    password
  };
  return http.post('/api/auth/password', data);
}

export function isPasswordUpdated() {
  return http.get('/api/auth/password');
}
