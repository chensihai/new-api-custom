import { API } from '@/helpers/api';

export const api = API;

export function getCommonHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Cache-Control': 'no-store',
  };

  try {
    const uid = window.localStorage.getItem('uid');
    if (uid && /^\d+$/.test(uid)) {
      headers['New-API-User'] = uid;
    }
  } catch {
    /* empty */
  }

  return headers;
}

export async function getSelf() {
  const res = await api.get('/api/user/self', {
    skipErrorHandler: true,
  });
  return res.data;
}

export async function getUserModels(): Promise<{
  success: boolean;
  message?: string;
  data?: string[];
}> {
  const res = await api.get('/api/user/models');
  return res.data;
}

export async function getUserGroups(): Promise<{
  success: boolean;
  message?: string;
  data?: Record<string, { desc: string; ratio: number | string }>;
}> {
  const res = await api.get('/api/user/self/groups');
  return res.data;
}

export async function getStatus() {
  const res = await api.get('/api/status');
  return res.data?.data;
}

export async function getNotice(): Promise<{
  success: boolean;
  message?: string;
  data?: string;
}> {
  const res = await api.get('/api/notice');
  return res.data;
}
