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
