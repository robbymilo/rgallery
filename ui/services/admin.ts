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
    const data = (await res.json()) as Record<string, unknown>;
    msg = (data['msg'] as string) || (data['error'] as string) || msg;
    throw new Error(msg);
  }
  return (await res.json()) as AdminData;
}
