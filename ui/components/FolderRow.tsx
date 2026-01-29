import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { getFolderContents } from '../services/folder';
import { Folder, MediaItem } from '../types';

import View from '../svg/view.svg?react';
import FolderIcon from '../svg/foldericon.svg?react';
import ChevronRight from '../svg/chevron-right.svg?react';
import ChevronDown from '../svg/chevron-down.svg?react';

interface FolderRowProps {
  folder: Folder;
}

const FolderRow: React.FC<FolderRowProps> = ({ folder }) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const [subFolders, setSubFolders] = useState<Folder[]>(folder.folders || []);
  const [subFiles, setSubFiles] = useState<MediaItem[]>(folder.files || []);

  const handleToggle = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    if (isExpanded) {
      setIsExpanded(false);
      return;
    }

    setIsExpanded(true);
    const hasData =
      folder.folders !== undefined || folder.files !== undefined || subFolders.length > 0 || subFiles.length > 0;

    if (!hasData) {
      try {
        const data = await getFolderContents(folder.id);
        setSubFolders(data.folders);
        setSubFiles(data.files);
      } catch (err) {
        console.error('Failed to load subfolder content', err);
      }
    }
  };

  return (
    <div className="dark:border-charcoal-800 border-b border-gray-100 last:border-0">
      {/* row header */}
      <div
        className={`group dark:hover:bg-charcoal-800/50 flex items-center justify-between p-2 transition-colors hover:bg-gray-50 ${isExpanded ? 'dark:bg-charcoal-800/30 bg-gray-50' : ''} `}
      >
        <div className="flex min-w-0 flex-1 items-center gap-4">
          <button
            onClick={handleToggle}
            className="hover:bg-primary-50 hover:text-primary-500 dark:hover:bg-primary-500/10 cursor-pointer rounded-md p-1 text-gray-400 transition-colors"
            title={isExpanded ? 'Collapse' : 'Expand'}
          >
            {isExpanded ? <ChevronDown /> : <ChevronRight />}
          </button>

          <div className="shrink-0">
            <div className="dark:bg-charcoal-700 bg-primary-50 text-primary-500 dark:text-primary-400 flex h-10 w-10 items-center justify-center rounded-lg">
              <FolderIcon />
            </div>
          </div>

          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-3">
              <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-white">{folder.name}</h3>
              {folder.itemCount > 0 && (
                <span className="dark:bg-charcoal-700 dark:text-charcoal-400 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
                  {folder.itemCount}
                </span>
              )}
            </div>

            {folder.previewImages && folder.previewImages.length > 0 && (
              <div className="mt-1 flex items-center gap-2">
                {folder.previewImages.slice(0, 5).map((src, i) => (
                  <div
                    key={i}
                    className="dark:border-charcoal-700 relative h-8 w-8 overflow-hidden rounded border border-gray-100"
                  >
                    <img src={src} alt="" className="h-full w-full object-cover" />
                  </div>
                ))}
                {(() => {
                  const total = folder.itemCount ?? folder.previewImages.length ?? 0;
                  const shown = Math.min(5, folder.previewImages.length);
                  const remaining = Math.max(0, total - shown);
                  return remaining > 0 ? (
                    <div className="dark:bg-charcoal-800 flex h-8 w-8 items-center justify-center rounded bg-gray-100 text-[10px] font-medium text-gray-500">
                      +{remaining}
                    </div>
                  ) : null;
                })()}
              </div>
            )}
          </div>
        </div>

        {folder.previewImages && folder.previewImages.length > 0 && (
          <div className="ml-4 flex items-center">
            <Link
              to={`/?folder=${folder.id}`}
              className="bg-primary-50 text-primary-600 hover:bg-primary-100 dark:bg-primary-500/10 dark:text-primary-400 dark:hover:bg-primary-500/20 flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors"
            >
              <View />
              View folder
            </Link>
          </div>
        )}
      </div>

      {isExpanded && (
        <div className="dark:border-charcoal-700 border-primary-100 my-2 ml-6 border-l-2 pl-6 md:ml-5.5 md:pl-8">
          <>
            {subFolders.length > 0 && (
              <div className="flex flex-col">
                {subFolders.map((subFolder) => (
                  <FolderRow key={subFolder.id} folder={subFolder} />
                ))}
              </div>
            )}

            {subFiles.length > 0 && (
              <div className="my-4">
                <h4 className="mb-3 text-xs font-bold tracking-wider text-gray-400 uppercase">Files</h4>
                <div className="grid grid-cols-2 gap-4 pr-4 sm:grid-cols-3 md:grid-cols-4">
                  {subFiles.slice(0, 4).map((file) => (
                    <Link
                      key={file.id}
                      to={`/media/${file.id}`}
                      className="group relative block aspect-video overflow-hidden rounded-lg"
                    >
                      <img src={file.thumbnailUrl} alt={file.title} className="h-full w-full object-cover" />
                    </Link>
                  ))}
                </div>
              </div>
            )}
          </>
        </div>
      )}
    </div>
  );
};

export default FolderRow;
