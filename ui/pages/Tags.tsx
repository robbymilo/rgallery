import React, { useEffect, useState, useMemo } from 'react';
import { useAuth } from '../context/AuthContext';
import { Link, useNavigate } from 'react-router-dom';
import { fetchTags, Tag as ServiceTag } from '../services/tags';
import Loading from '../components/Loading';
import Error from '../components/Error';

const Tags: React.FC = () => {
  const [tags, setTags] = useState<ServiceTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');

  const { logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    let mounted = true;

    const getTags = async () => {
      try {
        setLoading(true);
        const data = await fetchTags();
        if (mounted) {
          setTags(data);
          setError(null);
        }
      } catch (err: unknown) {
        if (!mounted) return;
        setError((err as Error)?.message || 'Failed to load tags.');
      } finally {
        if (mounted) setLoading(false);
      }
    };

    getTags();

    return () => {
      mounted = false;
    };
  }, [logout, navigate]);

  const filteredTags = useMemo(() => {
    return tags.filter((t) => t.value.toLowerCase().includes(searchTerm.toLowerCase()));
  }, [tags, searchTerm]);

  if (loading) {
    return <Loading />;
  }

  if (error) {
    return <Error error={error} />;
  }

  return (
    <div className="mx-auto w-[90vw] flex-1 py-8 md:w-[80vw]">
      <div className="mb-8 flex flex-col justify-between gap-4 md:flex-row md:items-end">
        <div>
          <h1>Tags</h1>
        </div>
      </div>

      <div className="dark:border-charcoal-800 dark:bg-charcoal-900 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap gap-2">
          {filteredTags.map((tag) =>
            tag.value !== '' ? (
              <Link
                key={tag.key}
                to={`/?tag=${encodeURIComponent(tag.key)}`}
                className="group dark:border-charcoal-700 dark:bg-charcoal-800/80 dark:text-charcoal-300 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700 dark:hover:border-primary-500/30 dark:hover:bg-primary-500/20 dark:hover:text-primary-300 inline-flex items-center rounded-md border border-gray-200 bg-gray-50 px-3 py-1.5 text-sm font-medium text-gray-700 transition-all duration-200"
              >
                <span>{tag.value}</span>
                {typeof tag.count === 'number' && (
                  <span className="dark:bg-charcoal-700 dark:text-charcoal-400 group-hover:bg-primary-100 group-hover:text-primary-600 dark:group-hover:bg-primary-500/30 dark:group-hover:text-primary-200 ml-2 rounded-full bg-gray-200 px-1.5 py-0.5 font-mono text-[10px] text-gray-500 transition-colors">
                    {tag.count}
                  </span>
                )}
              </Link>
            ) : null
          )}
        </div>
      </div>
    </div>
  );
};

export default Tags;
