import React, { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import Loading from '../components/Loading';
import Error from '../components/Error';
import { getScanErrors, resetScanError, ScanError } from '../services/admin';

const pageSize = 25;

const AdminScanErrors: React.FC = () => {
  const [items, setItems] = useState<ScanError[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await getScanErrors(page, pageSize);
      setItems(response.items);
      setTotal(response.meta.total);
      setError(null);
    } catch (err) {
      setError(err instanceof globalThis.Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => { void load(); }, [load]);

  const reset = async (path?: string) => {
    if (!window.confirm(path ? `Reset scan error for ${path}?` : 'Reset all scan errors?')) return;
    try {
      await resetScanError(path);
      if (!path && page !== 1) setPage(1);
      await load();
    } catch (err) {
      setError(err instanceof globalThis.Error ? err.message : String(err));
    }
  };

  if (loading && items.length === 0) return <Loading />;
  if (error && items.length === 0) return <Error error={error} />;

  const pages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="mx-auto w-[90vw] flex-1 space-y-6 py-8 md:w-[80vw]">
      <div className="flex items-center justify-between">
        <div>
          <Link to="/admin" className="text-primary-400 text-sm hover:underline">← Admin</Link>
          <h1>Scan errors</h1>
          <p className="dark:text-charcoal-400 text-sm text-gray-600">{total} item{total === 1 ? '' : 's'} skipped after an error.</p>
        </div>
        <button onClick={() => void reset()} disabled={total === 0} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50">Reset all</button>
      </div>
      {error && <div className="rounded border border-red-300 bg-red-100 p-3 text-sm text-red-800">{error}</div>}
      <section className="dark:border-charcoal-700 dark:bg-charcoal-800/50 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead><tr className="dark:border-charcoal-700 dark:text-charcoal-400 border-b border-gray-200 text-gray-500"><th className="p-3">Path</th><th className="p-3">Last error</th><th className="p-3">Error</th><th className="p-3">Action</th></tr></thead>
            <tbody className="dark:divide-charcoal-700/50 divide-y divide-gray-100">
              {items.map((item) => <tr key={item.path}><td className="max-w-[30rem] p-3 font-medium break-all">{item.path}</td><td className="whitespace-nowrap p-3 text-xs">{new Date(item.modified).toLocaleString()}</td><td className="max-w-[30rem] p-3 break-words">{item.error}</td><td className="p-3"><button onClick={() => void reset(item.path)} className="text-red-600 hover:underline dark:text-red-400">Reset</button></td></tr>)}
              {items.length === 0 && <tr><td colSpan={4} className="p-6 text-center text-gray-500">No scan errors.</td></tr>}
            </tbody>
          </table>
        </div>
        {pages > 1 && <div className="flex items-center justify-between border-t border-gray-200 p-3 text-sm dark:border-charcoal-700"><button onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={page === 1} className="rounded border px-3 py-1 disabled:opacity-50">Previous</button><span>Page {page} of {pages}</span><button onClick={() => setPage((value) => Math.min(pages, value + 1))} disabled={page === pages} className="rounded border px-3 py-1 disabled:opacity-50">Next</button></div>}
      </section>
    </div>
  );
};

export default AdminScanErrors;
