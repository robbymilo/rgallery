import { NotificationsResponse } from '../types';

export async function getNotifications(page: number, pageSize: number): Promise<NotificationsResponse> {
  const res = await fetch(`/api/notifications?page=${page}&pageSize=${pageSize}`);
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
  return res.json();
}

export async function markNotificationAsRead(id: number): Promise<void> {
  const res = await fetch(`/api/notifications/${id}`, { method: 'PATCH' });
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
}

export async function clearNotifications(): Promise<void> {
  const res = await fetch('/api/notifications/clear', { method: 'POST' });
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
}
