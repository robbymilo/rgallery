export function parseSearchTokens(raw: string): {
  searchQuery: string;
  tag?: string;
  camera?: string;
  lens?: string;
  software?: string;
  folder?: string;
  focallength35?: number;
} {
  // Extract a single token:value pattern at the end of the string
  const tokenPattern = /\b(tag|camera|lens|software|folder|focallength35):(.+?)$/i;
  const match = raw.match(tokenPattern);

  if (!match) {
    return {
      searchQuery: raw.trim(),
      tag: undefined,
      camera: undefined,
      lens: undefined,
      software: undefined,
      folder: undefined,
      focallength35: undefined,
    };
  }

  const key = match[1].toLowerCase();
  const value = match[2].trim();
  const searchQuery = raw.slice(0, match.index).trim();

  return {
    searchQuery,
    tag: key === 'tag' ? value : undefined,
    camera: key === 'camera' ? value : undefined,
    lens: key === 'lens' ? value : undefined,
    software: key === 'software' ? value : undefined,
    folder: key === 'folder' ? value : undefined,
    focallength35: key === 'focallength35' ? parseInt(value, 10) : undefined,
  };
}
