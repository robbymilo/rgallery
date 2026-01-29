export type GearStat = {
  name: string;
  count: number;
};

export type GearResponse = {
  cameras: GearStat[];
  lenses: GearStat[];
  focalLengths: GearStat[];
  software: GearStat[];
};

function normalizeStats(arr: { name: string; total: number }[]): GearStat[] {
  return arr.map(({ name, total }) => ({ name, count: total }));
}

export async function getGear(): Promise<GearResponse> {
  const res = await fetch('/api/gear');
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
  const data = await res.json();
  return {
    cameras: normalizeStats(data.camera || []),
    lenses: normalizeStats(data.lens || []),
    focalLengths: normalizeStats(data.focalLength35 || []),
    software: normalizeStats(data.software || []),
  };
}
