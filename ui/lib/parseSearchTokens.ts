export function parseSearchTokens(raw: string): {
  searchQuery: string;
  tag?: string;
  camera?: string;
  lens?: string;
  software?: string;
  folder?: string;
  focallength35?: number;
} {
  const result = {
    searchQuery: '',
    tag: undefined as string | undefined,
    camera: undefined as string | undefined,
    lens: undefined as string | undefined,
    software: undefined as string | undefined,
    folder: undefined as string | undefined,
    focallength35: undefined as number | undefined,
  };
  const tokenPattern =
    /(?:^|\s)(tag|camera|lens|software|folder|focallength35):\s*(.*?)(?=\s+(?:tag|camera|lens|software|folder|focallength35):|$)/gi;
  result.searchQuery = raw
    .replace(tokenPattern, (_, key: string, value: string) => {
      const normalized = key.toLowerCase() as Exclude<keyof typeof result, 'searchQuery'>;
      const text = value.trim();
      if (normalized === 'focallength35') {
        const number = Number(text);
        result.focallength35 = text && Number.isFinite(number) && number > 0 ? number : undefined;
      } else {
        result[normalized] = text || undefined;
      }
      return '';
    })
    .trim();
  return result;
}
