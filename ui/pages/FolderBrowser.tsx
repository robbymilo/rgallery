import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getFolderContents } from '../services/folder';
import { Folder } from '../types';
import FolderRow from '../components/FolderRow';
import Loading from '../components/Loading';
import Error from '../components/Error';

const FolderBrowser: React.FC = () => {
  const params = useParams();
  // Decode param URLs ex 'root/Folder%201'
  const folderPath = params['*'] ? decodeURIComponent(params['*'] as string) : 'root';
  const navigate = useNavigate();
  const [folders, setFolders] = useState<Folder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);

    const fetchData = async () => {
      try {
        const data = await getFolderContents(folderPath);
        setFolders(data.folders);
        setError(null);
      } catch (err: any) {
        console.error(err);
        setError(err.message || 'Failed to load folders.');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [folderPath]);

  if (loading) {
    return <Loading />;
  }

  if (error) {
    return <Error error={error} />;
  }

  return (
    <div className="mx-auto w-[90vw] flex-1 py-8 md:w-[80vw]">
      <div className="animate-fade-in min-h-screen pb-12">
        <header className="mb-8">
          <h1>Folders</h1>
        </header>

        <section className="dark:border-charcoal-800 dark:bg-charcoal-900 mb-10 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
          {folders.length === 0 ? (
            <div className="dark:text-charcoal-500 p-8 text-center text-gray-500 italic">
              No subfolders found in this directory.
            </div>
          ) : (
            <div className="dark:divide-charcoal-800 divide-y divide-gray-100">
              {folders.map((folder) => (
                <FolderRow key={folder.id} folder={folder} />
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
};

export default FolderBrowser;
