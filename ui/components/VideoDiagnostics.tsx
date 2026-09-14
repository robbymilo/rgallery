import React, { useEffect, useState } from 'react';

interface Diagnostics {
  encoder: string;
  device?: string;
  rateControl?: string;
  fallback?: string;
  workers: number;
  cacheBytes: number;
  cacheLimitBytes: number;
  lastEncodeSeconds: number;
  lastEncodeSpeed: number;
  jobs: { id: string; state: string; seconds: number }[];
  recent: { id: string; state: string; seconds: number; error?: string }[];
}

// Displays encoder status, jobs, and cache usage.
const VideoDiagnostics: React.FC = () => {
  const [data, setData] = useState<Diagnostics | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    // Refreshes diagnostics and schedules the next update.
    const refresh = async () => {
      try {
        const response = await fetch('/api/transcode/diagnostics', { signal: controller.signal, cache: 'no-store' });
        if (!response.ok) throw new Error('Unable to load video diagnostics.');
        setData(await response.json());
        setError('');
      } catch (e) {
        if (!controller.signal.aborted) setError((e as Error).message);
      }
      if (!controller.signal.aborted) timer = setTimeout(refresh, 5000);
    };
    void refresh();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, []);

  return (
    <section className="dark:border-charcoal-700 dark:bg-charcoal-800/50 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
      <h2 className="mb-4 text-xl font-semibold">Video playback</h2>
      {error && <p role="alert">{error}</p>}
      {!data && !error && <p role="status">Checking video capabilities…</p>}
      {data && (
        <div className="space-y-3 text-sm">
          <p>
            Encoder: <strong>{data.encoder}</strong>
            {data.device && ` · ${data.device}`}
            {data.rateControl && ` · ${data.rateControl}`}
          </p>
          {data.fallback && <p>{data.fallback}</p>}
          <p>
            {data.workers} workers · {data.jobs.length} active or queued jobs · Cache{' '}
            {(data.cacheBytes / 1048576).toFixed(1)} / {(data.cacheLimitBytes / 1048576).toFixed(0)} MiB
          </p>
          {data.lastEncodeSeconds > 0 && (
            <p>
              Last encode: {data.lastEncodeSeconds.toFixed(2)} s · {data.lastEncodeSpeed.toFixed(1)}× playback speed
            </p>
          )}
          {(data.jobs.length > 0 || data.recent.length > 0) && (
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr>
                    <th className="py-2">Job</th>
                    <th>Status</th>
                    <th>Elapsed</th>
                  </tr>
                </thead>
                <tbody>
                  {[...data.jobs, ...data.recent.slice(0, 5)].map((job, i) => (
                    <tr key={`${job.id}-${i}`}>
                      <td className="py-1">{job.id}</td>
                      <td>{job.state}</td>
                      <td>{job.seconds.toFixed(2)} s</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </section>
  );
};

export default VideoDiagnostics;
