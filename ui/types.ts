export interface User {
  username: string;
  role: 'admin' | 'viewer';
}

export interface ApiKey {
  id: string;
  name: string;
  key: string;
  createdAt: string;
}

export interface PreviewImage {
  url: string;
  path: string;
}

export interface Folder {
  id: string;
  name: string;
  itemCount: number;
  path?: string;
  previewImages?: PreviewImage[];
  folders?: Folder[];
  files?: MediaItem[];
}

export type RawMedia = {
  hash: number;
  path: string;
  width?: number;
  height?: number;
  type?: string;
  [k: string]: unknown;
};

export type RawFolder = {
  path: string;
  name?: string;
  imageCount?: number;
  total?: number;
  media?: RawMedia[];
  children?: RawFolder[];
  folders?: RawFolder[];
  [k: string]: unknown;
};

export interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

export interface ApiError {
  status: number;
  message: string;
}

export interface Photo {
  id: string;
  url: string;
  width: number;
  height: number;
  aspectRatio: number; // width / height
  date: Date;
  type: 'image' | 'video';
  color: string; // Placeholder color
  path: string;
}

// API Types matching the requested JSON structure
export interface ApiPhoto {
  id: number;
  w: number;
  h: number;
  c: string;
  t: string;
  d: string; // "YYYY-MM-DD"
  path: string;
}

export interface Notification {
  id: number;
  username: string;
  message: string;
  is_read: boolean;
  created_at: string;
}

export interface ApiTimelineItem {
  date: string;
  count: number;
}

export interface TimelineResponse {
  meta: {
    total: number;
    pagesize: number;
    nextCursor: string; // "1000", "2000" etc.
  };
  timeline: ApiTimelineItem[];
  photos: ApiPhoto[];
}

// Layout Node Types
export enum NodeType {
  DATE_HEADER = 'DATE_HEADER',
  PHOTO_ROW = 'PHOTO_ROW',
}

export interface LayoutNode {
  type: NodeType;
  id: string;
  height: number;
  top: number; // Calculated absolute position
}

export interface DateHeaderNode extends LayoutNode {
  type: NodeType.DATE_HEADER;
  date: string;
  dateObj: Date;
}

export interface PhotoRowNode extends LayoutNode {
  type: NodeType.PHOTO_ROW;
  photos: Photo[];
  rowHeight: number;
  layoutWidths: number[];
}

export type VirtualItem = DateHeaderNode | PhotoRowNode;

export interface GearStat {
  name: string;
  count: number;
}

export interface GearResponse {
  cameras: GearStat[];
  lenses: GearStat[];
  focalLengths: GearStat[];
  software: GearStat[];
}

export interface Tag {
  id: string;
  name: string;
  count: number;
  thumbnailUrl: string;
}

export interface MediaSubject {
  key: string;
  value: string;
}

export interface MediaItem {
  path?: string;
  subjects?: MediaSubject[];
  hash: number;
  width?: number;
  height?: number;
  ratio?: number;
  padding?: number;
  date?: string;
  modified?: string;
  folder?: string;
  srcset?: string;
  rating?: number;
  shutterspeed?: string;
  aperture?: number;
  iso?: number;
  lens?: string;
  camera?: string;
  focallength?: number;
  altitude?: number;
  latitude?: number;
  longitude?: number;
  type?: 'image' | 'video' | string;
  focusDistance?: number;
  focalLength35?: number;
  color?: string;
  location?: string;
  description?: string;
  title?: string;
  software?: string;
  offset?: number;
  // UI
  id?: string;
  thumbnailUrl?: string;
}

export interface MediaNeighbor {
  hash: number;
  color: string;
  type: string;
  path: string;
  width: number;
  height: number;
  srcset: string;
}

export interface MediaResponse {
  media: MediaItem;
  previous: MediaNeighbor[];
  next: MediaNeighbor[];
  collection: string;
  slug: string;
}

export interface ApiResponse {
  media: MediaItem;
  previous: MediaItem[];
  next: MediaItem[];
  collection: string;
  slug: string;
}

export enum ViewMode {
  NORMAL = 'NORMAL',
  FULLSCREEN = 'FULLSCREEN',
}

export interface Memory {
  key: number;
  value: string;
  media: MediaItem[];
  total: number;
}

export enum MediaType {
  IMAGE = 'Images',
  VIDEO = 'Videos',
}

export type SortOption = 'date-desc' | 'date-asc' | 'modified-desc' | 'modified-asc';

export interface FilterState {
  searchQuery: string;
  minRating: number | 0 | 1 | 2 | 3 | 4 | 5;
  tag?: string;
  folder?: string;
  camera?: string;
  lens?: string;
  software?: string;
  focallength35?: number;
  mediaType: 'image' | 'video' | 'all';
  sortBy: SortOption;
}

export interface TimelineFilters {
  term?: string;
  rating?: number;
  tag?: string;
  type?: 'image' | 'video';
  orderby?: 'date' | 'modified';
  direction?: 'asc' | 'desc';
  camera?: string;
  lens?: string;
  folder?: string;
  subject?: string;
  software?: string;
  focallength35?: number;
}
