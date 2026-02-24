import { Folder, RawMedia, RawFolder, MediaItem } from '../types';

async function getFolders(): Promise<RawFolder> {
  const res = await fetch('/api/folders');
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
  return (await res.json()) as RawFolder;
}

// recursively find folder by path
function findFolderByPath(node: RawFolder | null, path: string): RawFolder | null {
  if (!node) return null;
  if (path === 'root' || path === '' || !path) return node;
  if (node.path === path) return node;

  const children = node.folders || node.children || [];
  for (const child of children) {
    const found = findFolderByPath(child, path);
    if (found) return found;
  }
  return null;
}

const mapToMediaItem = (m: RawMedia): MediaItem => ({
  id: m.hash.toString(),
  title: m.path.split('/').pop() || '',
  thumbnailUrl: `/api/img/${m.hash}/400`,
  path: m.path,
  hash: m.hash,
  width: m.width || 0,
  height: m.height || 0,
  type: (m.type as MediaItem['type']) || 'image',
});

function mapToFolder(child: RawFolder): Folder {
  return {
    id: child.path,
    name: child.name || child.path.split('/').pop() || child.path,
    itemCount: child.imageCount || child.total || 0,
    previewImages: (child.media || []).slice(0, 5).map((m: RawMedia) => ({
      url: `/api/img/${m.hash}/400`,
      path: m.path,
    })),
    folders: (child.children || child.folders || []).map(mapToFolder),
    files: (child.media || []).map(mapToMediaItem),
  };
}

export async function getFolderContents(
  folderPath: string = 'root'
): Promise<{ folders: Folder[]; files: MediaItem[] }> {
  const root = await getFolders();

  let folders: Folder[] = [];
  let files: MediaItem[] = [];

  if (folderPath === 'root' || folderPath === '' || !folderPath) {
    folders = (root.folders || []).map(mapToFolder);
    files = [];
  } else {
    const folder = findFolderByPath(root, folderPath);
    if (!folder) throw new Error('Folder not found');
    folders = (folder.children || folder.folders || []).map(mapToFolder);
    files = (folder.media || []).map(mapToMediaItem);
  }

  return { folders, files };
}
