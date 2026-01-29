import React, { useState, useEffect } from 'react';
import { useWebSocket } from '../context/WebSocketContext';
import Copy from '../svg/copy.svg?react';
import Refresh from '../svg/refresh.svg?react';
import Loading from '../components/Loading';
import Error from '../components/Error';
import { getAdmin, AdminData, AdminApiKey } from '../services/admin';
import { User } from '../types';

enum ScanDescriptions {
  Scan = 'Scan for new and modified items and remove deleted items.',
  Metadata = 'Rescan all existing items without recreating thumbnails.',
  Deep = 'Rescan all existing items and recreate thumbnails.',
  Thumbnail = 'Check for missing thumbnails and generate any missing ones.',
}

const Admin: React.FC = () => {
  const { isScanInProgress } = useWebSocket();
  const [selectedScanType, setSelectedScanType] = useState<'scan' | 'metadata' | 'deep' | 'thumbnail'>('scan');

  // Data State
  const [adminData, setAdminData] = useState<AdminData | null>(null);
  const [loading, setLoading] = useState(true);
  const [notification, setNotification] = useState<{ msg: string; type: 'success' | 'error' } | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Scan State
  const [scanState, setScanState] = useState<{ type: string; loading: boolean }>({ type: '', loading: false });

  // New User State
  const [newUser, setNewUser] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState<'admin' | 'viewer'>('viewer');

  // API Key State
  const [newApiKeyName, setNewApiKeyName] = useState('');
  const [createdApiKey, setCreatedApiKey] = useState<string | null>(null);

  const handleScan = async () => {
    // Don't start new scan if one is already in progress
    if (isScanInProgress) {
      setNotification({ msg: 'Scan already in progress', type: 'error' });
      return;
    }

    setScanState({ type: selectedScanType, loading: true });
    try {
      const res = await fetch(`/api/scan?type=${selectedScanType}`, { credentials: 'include' });
      const data = await res.json().catch(() => null);
      if (res.ok) {
      } else {
        setNotification({ msg: data?.msg || 'Scan failed', type: 'error' });
      }
    } catch {
      setNotification({ msg: 'Scan failed', type: 'error' });
    } finally {
      setScanState({ type: '', loading: false });
    }
  };

  const handleCancel = async () => {
    try {
      const res = await fetch('/api/scan/cancel', { method: 'POST', credentials: 'include' });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setNotification({ msg: data?.msg || 'Failed to cancel scan', type: 'error' });
      }
      // WebSocket handles success notifications
    } catch {
      setNotification({ msg: 'Failed to cancel scan', type: 'error' });
    }
  };

  // Sync scan loading state with WebSocket status
  useEffect(() => {
    if (!isScanInProgress) {
      setScanState({ type: '', loading: false });
    }
  }, [isScanInProgress]);

  const handleAddUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newUser.trim() || !newPassword.trim()) return;
    try {
      const res = await fetch('/api/user/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: `username=${encodeURIComponent(newUser)}&password=${encodeURIComponent(newPassword)}&role=${newRole}`,
      });
      if (res.ok) {
        setNotification({ msg: 'User added.', type: 'success' });
        setNewUser('');
        setNewPassword('');
        setNewRole('viewer');
        fetchAdminData();
        // Redirect if the current user is 'admin'
        if (typeof adminData?.UserName === 'string' && adminData.UserName.toLowerCase() === 'admin') {
          window.location.href = '/';
        }
      } else {
        setNotification({ msg: 'Failed to add user', type: 'error' });
      }
    } catch {
      setNotification({ msg: 'Failed to add user', type: 'error' });
    }
  };

  const handleCreateApiKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newApiKeyName.trim()) return;
    try {
      const res = await fetch('/api/keys/create', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: `name=${encodeURIComponent(newApiKeyName)}`,
      });
      if (res.ok) {
        const data = await res.json();
        // Show the key value once
        setCreatedApiKey(data.key || data.Key?.Key || '');
        setNotification({ msg: 'API Key created.', type: 'success' });
        setNewApiKeyName('');
        fetchAdminData();
      } else {
        setNotification({ msg: 'Failed to create API key', type: 'error' });
      }
    } catch {
      setNotification({ msg: 'Failed to create API key', type: 'error' });
    }
  };

  const deleteApiKey = async (name: string) => {
    if (!window.confirm(`Delete API key "${name}"?`)) return;
    try {
      const res = await fetch('/api/keys/delete', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: `name=${encodeURIComponent(name)}`,
      });
      if (res.ok) {
        setNotification({ msg: 'API Key deleted.', type: 'success' });
        fetchAdminData();
      } else {
        setNotification({ msg: 'Failed to delete API key', type: 'error' });
      }
    } catch {
      setNotification({ msg: 'Failed to delete API key', type: 'error' });
    }
  };

  useEffect(() => {
    fetchAdminData();
  }, []);

  const fetchAdminData = async () => {
    setLoading(true);
    try {
      const data = await getAdmin();
      setAdminData(data);
      setError(null);
    } catch (e) {
      setError((e as Error)?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <Loading />;
  }

  if (error) {
    return <Error error={error} />;
  }

  if (!adminData || adminData.UserRole !== 'admin') {
    return <Error error="You do not have permission to view this page." />;
  }

  const users: User[] = adminData.Users || [];
  const apiKeys: AdminApiKey[] = adminData.Keys || [];

  return (
    <div className="mx-auto w-[90vw] flex-1 space-y-8 py-8 md:w-[80vw]">
      <div className="flex flex-col justify-between md:flex-row md:items-center">
        <div>
          <h1>Admin</h1>
        </div>
      </div>

      {/* Notification toast */}
      {notification && (
        <div
          className={`animate-bounce-in fixed right-4 bottom-4 z-50 flex items-center gap-3 rounded-lg border p-4 shadow-xl ${notification.type === 'success' ? 'border-green-300 bg-green-100 text-green-800 dark:border-green-700 dark:bg-green-900/90 dark:text-green-100' : 'border-red-300 bg-red-100 text-red-800 dark:border-red-700 dark:bg-red-900/90 dark:text-red-100'}`}
        >
          {notification.msg}
        </div>
      )}

      {/* Scan Actions */}
      <section className="dark:border-charcoal-700 dark:bg-charcoal-800/50 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-4 text-xl font-semibold text-gray-800 dark:text-white">Library</h2>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            {(['scan', 'metadata', 'deep', 'thumbnail'] as const).map((type) => (
              <label key={type} className="flex items-start gap-2">
                <input
                  type="radio"
                  name="scanType"
                  value={type}
                  checked={selectedScanType === type}
                  onChange={() => setSelectedScanType(type)}
                  className="dark:border-charcoal-700 dark:bg-charcoal-900 focus:ring-primary-500 text-primary-600 mt-1 rounded border-gray-300 bg-white"
                />
                <div>
                  <span className="font-medium text-gray-800 capitalize dark:text-white">{type}</span>
                  <p className="dark:text-charcoal-400 text-sm text-gray-600">
                    {ScanDescriptions[(type.charAt(0).toUpperCase() + type.slice(1)) as keyof typeof ScanDescriptions]}
                  </p>
                </div>
              </label>
            ))}
          </div>
          {!isScanInProgress ? (
            <button
              onClick={handleScan}
              className="bg-primary-600 hover:bg-primary-700 flex w-fit items-center gap-1 rounded-lg px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors"
            >
              <Refresh className="h-5 w-5" />
              Start scan
            </button>
          ) : (
            <button
              onClick={handleCancel}
              className="flex w-fit items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-red-700"
            >
              <Refresh className="h-5 w-5 animate-spin" />
              Cancel scan
            </button>
          )}
        </div>
      </section>

      {/* Users */}
      <section className="dark:border-charcoal-700 dark:bg-charcoal-800/50 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-xl font-semibold text-gray-800 dark:text-white">
            User management
          </h2>
          <span className="dark:border-charcoal-700 dark:bg-charcoal-800 dark:text-charcoal-400 rounded border border-gray-200 bg-gray-100 px-2 py-1 text-xs text-gray-500">
            {users.length} {users.length === 1 ? 'user' : 'users'}
          </span>
        </div>
        <div className="mb-6 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="dark:border-charcoal-700 dark:text-charcoal-400 border-b border-gray-200 text-gray-500">
                <th className="pb-3 pl-2">Username</th>
                <th className="pb-3">Role</th>
              </tr>
            </thead>
            <tbody className="dark:divide-charcoal-700/50 divide-y divide-gray-100">
              {users.map((u: User, idx: number) => (
                <tr key={idx} className="group dark:hover:bg-charcoal-700/30 transition-colors hover:bg-gray-50">
                  <td className="dark:text-charcoal-200 py-3 pl-2 font-medium text-gray-900">{u.username}</td>
                  <td className="py-3">
                    <span
                      className={`rounded border px-2 py-0.5 text-xs font-medium ${u.role === 'admin' ? 'border-primary-200 bg-primary-100 text-primary-700 dark:border-primary-500/20 dark:bg-primary-500/10 dark:text-primary-300' : 'dark:border-charcoal-600/20 dark:bg-charcoal-600/10 dark:text-charcoal-300 border-gray-200 bg-gray-100 text-gray-600'}`}
                    >
                      {u.role?.toUpperCase()}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* Add user */}
        <form
          onSubmit={handleAddUser}
          className="dark:border-charcoal-700 flex flex-wrap gap-2 border-t border-gray-200 pt-4"
        >
          <input
            type="text"
            placeholder="New username..."
            value={newUser}
            onChange={(e) => setNewUser(e.target.value)}
            className="dark:border-charcoal-700 dark:bg-charcoal-900 focus:ring-primary-500 min-w-[200px] flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:outline-none dark:text-white"
          />
          <input
            type="password"
            placeholder="Password..."
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            className="dark:border-charcoal-700 dark:bg-charcoal-900 focus:ring-primary-500 min-w-[200px] flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:outline-none dark:text-white"
          />
          <select
            value={newRole}
            onChange={(e) => setNewRole(e.target.value as 'admin' | 'viewer')}
            className="dark:border-charcoal-700 dark:bg-charcoal-900 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-900 dark:text-white"
          >
            <option value="viewer">Viewer</option>
            <option value="admin">Admin</option>
          </select>
          <button
            type="submit"
            disabled={!newUser.trim() || !newPassword.trim()}
            className="bg-primary-600 hover:bg-primary-700 flex items-center gap-1 rounded-lg px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors disabled:opacity-50"
          >
            Add
          </button>
        </form>
      </section>

      {/* API keys */}
      <section className="dark:border-charcoal-700 dark:bg-charcoal-800/50 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-xl font-semibold text-gray-800 dark:text-white">API keys</h2>
          <span className="dark:border-charcoal-700 dark:bg-charcoal-800 dark:text-charcoal-400 rounded border border-gray-200 bg-gray-100 px-2 py-1 text-xs text-gray-500">
            {apiKeys.length} {apiKeys.length === 1 ? 'key' : 'keys'}
          </span>
        </div>
        {/* Show created key value once */}
        {createdApiKey && (
          <div className="bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-100 mb-4 flex flex-col gap-2 rounded p-3">
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm break-all">{createdApiKey}</span>
              <button
                onClick={() => {
                  navigator.clipboard.writeText(createdApiKey);
                  setNotification({ msg: 'Copied', type: 'success' });
                }}
                className="text-primary-700 hover:bg-primary-200 dark:text-primary-200 dark:hover:bg-primary-800 ml-2 rounded px-2 py-1 text-xs"
                title="Copy to clipboard"
              >
                <Copy className="h-6 w-6" />
              </button>
            </div>
            <div className="text-primary-700 dark:text-primary-200 text-xs">
              This key will only be shown once. Please copy and store it securely.
            </div>
          </div>
        )}
        <div className="mb-6 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="dark:border-charcoal-700 dark:text-charcoal-400 border-b border-gray-200 text-gray-500">
                <th className="pb-3 pl-2">Name</th>
                <th className="pb-3">Created</th>
                <th className="pb-3">Actions</th>
              </tr>
            </thead>
            <tbody className="dark:divide-charcoal-700/50 divide-y divide-gray-100">
              {apiKeys.map((k: AdminApiKey, idx: number) => (
                <tr key={idx} className="group dark:hover:bg-charcoal-700/30 transition-colors hover:bg-gray-50">
                  <td className="py-3 pl-2">
                    <div className="dark:text-charcoal-200 font-medium text-gray-900">{k.name}</div>
                  </td>
                  <td className="dark:text-charcoal-400 py-3 text-xs text-gray-500">{k.created ?? k.createdAt}</td>
                  <td className="py-3">
                    <button
                      title="Delete API Key"
                      onClick={() => deleteApiKey(k.name)}
                      className="rounded p-1 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* Add API key form */}
        <form
          onSubmit={handleCreateApiKey}
          className="dark:border-charcoal-700 flex gap-2 border-t border-gray-200 pt-4"
        >
          <input
            type="text"
            placeholder="API Key Name..."
            value={newApiKeyName}
            onChange={(e) => setNewApiKeyName(e.target.value)}
            className="dark:border-charcoal-700 dark:bg-charcoal-900 focus:ring-primary-500 flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:outline-none dark:text-white"
          />
          <button
            type="submit"
            disabled={!newApiKeyName.trim()}
            className="bg-primary-600 hover:bg-primary-700 flex items-center gap-1 rounded-lg px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors disabled:opacity-50"
          >
            Create
          </button>
        </form>
      </section>
    </div>
  );
};

export default Admin;
