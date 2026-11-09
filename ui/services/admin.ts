import { User, ApiKey } from '../types';

export type AdminApiKey = {
  name: string;
  created?: string;
  createdAt?: string;
} & Record<string, unknown>;

export type AdminData = {
  UserRole?: string;
  UserName?: string;
  Users?: User[];
  Keys?: AdminApiKey[];
  [key: string]: unknown;
};

export type ScanError = {
  path: string;
  modified: string;
  error: string;
};

export type ScanErrorsResponse = {
  items: ScanError[];
  meta: { page: number; pageSize: number; total: number };
};

export async function getAdmin(): Promise<AdminData> {
  const res = await fetch('/api/admin', { credentials: 'include' });
  if (!res.ok) {
    let msg = `API Error: ${res.status}`;
    const contentType = res.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      const data = (await res.json().catch(() => null)) as Record<string, unknown> | null;
      msg = (data?.['msg'] as string) || (data?.['error'] as string) || msg;
    } else {
      const text = (await res.text()).trim();
      if (text) msg = text;
    }
    throw new Error(msg);
  }
  return (await res.json()) as AdminData;
}

export async function getScanErrors(page: number, pageSize = 25): Promise<ScanErrorsResponse> {
  const res = await fetch(`/api/admin/scan-errors?page=${page}&pageSize=${pageSize}`, { credentials: 'include' });
  if (!res.ok) throw new Error(`API Error: ${res.status}`);
  return (await res.json()) as ScanErrorsResponse;
}

export async function resetScanError(path?: string): Promise<void> {
  const query = path ? `?path=${encodeURIComponent(path)}` : '?all=true';
  const res = await fetch(`/api/admin/scan-errors${query}`, { method: 'DELETE', credentials: 'include' });
  if (!res.ok) throw new Error((await res.text()) || `API Error: ${res.status}`);
}
