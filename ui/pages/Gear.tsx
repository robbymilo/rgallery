import React, { useEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { Link, useNavigate } from 'react-router-dom';
import { getGear, GearStat, GearResponse } from '../services/gear';
import Loading from '../components/Loading';
import Error from '../components/Error';

const GearTable: React.FC<{
  title: string;
  queryKey: string;
  data: GearStat[];
  color: string;
}> = ({ title, queryKey, data, color }) => {
  const [expanded, setExpanded] = useState(false);

  const visibleCount = 10;
  const hasMore = data.length > visibleCount;
  const visible = hasMore ? data.slice(0, visibleCount) : data;
  const hidden = hasMore ? data.slice(visibleCount) : [];

  const rowHeight = 48; // matches px-4 py-3 + text sizing roughly
  const hiddenMaxHeight = expanded ? hidden.length * rowHeight : 0;

  return (
    <div className="dark:border-charcoal-700 dark:bg-charcoal-800 flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
      <div
        className={`dark:border-charcoal-700 flex items-center justify-between border-b border-gray-100 p-4 ${color} bg-opacity-5 dark:bg-opacity-10`}
      >
        <h3 className="flex items-center gap-2 font-semibold text-white">{title}</h3>
        <span className="dark:border-charcoal-700 dark:bg-charcoal-900 dark:text-charcoal-400 rounded border border-gray-100 bg-white px-2 py-1 text-xs font-medium text-gray-500 dark:text-white">
          {data.length} total
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="dark:bg-charcoal-800/50 dark:text-charcoal-400 bg-gray-50 font-medium text-gray-500">
            <tr>
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3 text-right">Count</th>
            </tr>
          </thead>

          <tbody className="dark:divide-charcoal-700/50 divide-y divide-gray-100">
            {visible.map((item, index) => (
              <tr key={index} className="dark:hover:bg-charcoal-700/20 transition-colors hover:bg-gray-50">
                <td className="dark:text-charcoal-200 max-w-[200px] truncate px-4 py-3 font-medium text-gray-900">
                  {item.name ? (
                    <Link to={`/?${queryKey}=${encodeURIComponent(item.name)}`} className="hover:underline">
                      {item.name}
                    </Link>
                  ) : (
                    <span className="text-gray-400 italic">Unknown</span>
                  )}
                </td>
                <td className="dark:text-charcoal-300 px-4 py-3 text-right font-mono text-gray-600">
                  {item.count.toLocaleString('en-GB')}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {hasMore && (
          <div className="overflow-hidden transition-[max-height] duration-300" style={{ maxHeight: hiddenMaxHeight }}>
            <table className="w-full text-left text-sm">
              <tbody className="dark:divide-charcoal-700/50 divide-y divide-gray-100">
                {hidden.map((item, idx) => (
                  <tr
                    key={visibleCount + idx}
                    className="dark:hover:bg-charcoal-700/20 transition-colors hover:bg-gray-50"
                  >
                    <td className="dark:text-charcoal-200 max-w-[200px] truncate px-4 py-3 font-medium text-gray-900">
                      {item.name ? (
                        <Link to={`/?${queryKey}=${encodeURIComponent(item.name)}`} className="hover:underline">
                          {item.name}
                        </Link>
                      ) : (
                        <span className="text-gray-400 italic">Unknown</span>
                      )}
                    </td>
                    <td className="dark:text-charcoal-300 px-4 py-3 text-right font-mono text-gray-600">
                      {item.count.toLocaleString('en-GB')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {hasMore && (
          <div className="dark:border-charcoal-700 flex justify-center border-t border-gray-100 px-4 py-3">
            <button
              onClick={() => setExpanded((s) => !s)}
              className="dark:bg-charcoal-800 dark:text-charcoal-200 dark:hover:bg-charcoal-700/60 rounded-md bg-gray-50 px-3 py-1 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border dark:border-transparent"
              aria-expanded={expanded}
            >
              {expanded ? `Show less` : `Show ${data.length - visibleCount} more`}
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

const Gear: React.FC = () => {
  const [stats, setStats] = useState<GearResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    let mounted = true;

    const fetchGear = async () => {
      try {
        setLoading(true);
        const data = await getGear();
        if (mounted) {
          setStats(data);
          setError(null);
        }
      } catch (err: any) {
        if (!mounted) return;
        setError(err && err.message ? err.message : 'Failed to load gear statistics.');
      } finally {
        if (mounted) setLoading(false);
      }
    };

    fetchGear();

    return () => {
      mounted = false;
    };
  }, [logout, navigate]);

  if (loading) {
    return <Loading />;
  }

  if (error || !stats) {
    return <Error error={error} />;
  }

  return (
    <div className="mx-auto w-[90vw] flex-1 py-8 md:w-[80vw]">
      <div className="mx-auto mb-8">
        <h1>Gear stats</h1>
      </div>

      <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
        {/* todo - enum */}
        <GearTable title="Cameras" queryKey="camera" data={stats.cameras} color="bg-primary-500" />
        <GearTable title="Lenses" queryKey="lens" data={stats.lenses} color="bg-primary-500" />
        <GearTable
          title="Focal Length (35mm equivalent)"
          queryKey="focallength35"
          data={stats.focalLengths}
          color="bg-primary-500"
        />
        <GearTable title="Software" queryKey="software" data={stats.software} color="bg-primary-500" />
      </div>
    </div>
  );
};

export default Gear;
