import React from 'react';
import { MediaItem } from '../types';
import Star from '../svg/star.svg?react';
import MapComponent from './MapComponent';

interface ExifPanelProps {
  media: MediaItem;
}

const DetailRow = ({
  label,
  value,
  fullWidth,
  title,
}: {
  label: string;
  value: string | number | undefined;
  fullWidth?: boolean;
  title?: string;
}) => {
  if (value === undefined || value === null || value === '') return null;
  return (
    <div
      className={`flex items-start justify-between rounded border-b border-zinc-300 px-1 py-1 transition-colors last:border-0 dark:border-zinc-800 dark:hover:bg-zinc-900/30`}
    >
      <span className="mt-0.5 mr-4 w-32 shrink-0 text-sm font-medium text-zinc-700 dark:text-zinc-500">{label}</span>
      <span
        className={`text-right font-mono text-sm text-zinc-900 dark:text-zinc-200 ${fullWidth ? 'break-all whitespace-normal' : 'break-words'}`}
        style={{ wordBreak: fullWidth ? 'break-all' : 'normal' }}
        title={title}
      >
        {value}
      </span>
    </div>
  );
};

const RatingRow = ({ rating }: { rating?: number }) => {
  if (rating === undefined || rating === null) return null;
  return (
    <div className="-mx-2 mt-2 flex flex-col gap-3 px-2 py-2">
      <span className="text-[10px] font-bold tracking-widest text-zinc-700 dark:text-zinc-500">Rating</span>
      <div className="flex items-center gap-2">
        <div className="flex gap-1.5">
          {[1, 2, 3, 4, 5].map((star) => (
            <Star
              key={star}
              className={`h-6 w-6 ${star <= rating ? 'fill-yellow-400 text-yellow-500' : 'text-zinc-400 dark:text-zinc-700'}`}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

const ColorRow = ({ color }: { color?: string }) => {
  if (!color) return null;
  return (
    <div className="-mx-3 flex justify-between border-b border-zinc-300 px-3 py-3 dark:border-zinc-800">
      <span className="w-32 text-sm font-medium text-zinc-700 dark:text-zinc-500">Dominant color</span>
      <div className="flex items-center gap-3">
        <span className="font-mono text-sm text-zinc-900 uppercase dark:text-zinc-200">{color}</span>
        <div
          className="h-6 w-6 rounded-md border border-zinc-400 shadow-sm dark:border-zinc-700"
          style={{ backgroundColor: color }}
        ></div>
      </div>
    </div>
  );
};

const ExifPanel: React.FC<ExifPanelProps> = ({ media }) => {
  // displayDate takes a local time and offset and returns a date string in RFC1123Z format.
  const displayDate = (d: string, o: number) => {
    // https://stackoverflow.com/questions/7403486/add-or-subtract-timezone-difference-to-javascript-date
    const targetTime = new Date(d);
    if (o) {
      const timeZoneFromDB = o / 60; //time zone value from database
      //get the timezone offset from local time in minutes
      const tzDifference = timeZoneFromDB * 60 + targetTime.getTimezoneOffset();
      //convert the offset to milliseconds, add to targetTime, and make a new Date
      const offsetTime = new Date(targetTime.getTime() + tzDifference * 60 * 1000);

      const dayString = new Intl.DateTimeFormat('en-GB', {
        weekday: 'short',
        year: 'numeric',
        month: 'long',
        day: '2-digit',
      }).format(offsetTime);

      const timeString = new Intl.DateTimeFormat('en-GB', {
        hour: 'numeric',
        minute: 'numeric',
        second: 'numeric',
        fractionalSecondDigits: 3,
        hour12: false,
      }).format(offsetTime);

      return `${dayString} ${timeString}`;
    } else {
      // the UTC offset of the photo is unknown, so display it as is.
      const dayString = new Intl.DateTimeFormat('en-GB', {
        weekday: 'short',
        year: 'numeric',
        month: 'long',
        day: '2-digit',
        timeZone: 'GMT',
      }).format(targetTime);

      const timeString = new Intl.DateTimeFormat('en-GB', {
        hour: 'numeric',
        minute: 'numeric',
        second: 'numeric',
        fractionalSecondDigits: 3,
        hour12: false,
        timeZone: 'GMT',
      }).format(targetTime);

      return `${dayString} ${timeString}`;
    }
  };

  const createdDate = displayDate(media.date, media.offset);
  const modifiedDate = displayDate(media.modified, 0);
  const megapixels = media.width && media.height ? ((media.width * media.height) / 1000000).toFixed(2) : undefined;

  return (
    <div className="mx-auto mt-8 mb-16 w-[90vw] bg-white md:w-[80vw] dark:bg-zinc-900">
      {/* Media header */}
      {media.title || media.description ? (
        <div className="w-[60vw] bg-white py-2 dark:bg-zinc-900">
          <h1 className="mb-3 text-3xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
            {media.title || 'Untitled'}
          </h1>
          {media.description && (
            <p className="text-md max-w-4xl text-zinc-600 dark:text-zinc-100">{media.description}</p>
          )}
        </div>
      ) : null}
      <div className="grid grid-cols-1 gap-8 py-8 md:grid-cols-2 lg:grid-cols-3">
        {/* Exif info */}
        <div className="space-y-2">
          <h3 className="mb-6 flex items-center gap-2 border-b border-zinc-300 pb-3 text-xl font-bold text-zinc-900 dark:border-zinc-700 dark:text-zinc-100">
            File details
          </h3>
          <DetailRow label="Created on" value={createdDate} title={media.date} />
          <DetailRow label="Modified on" value={modifiedDate} />
          <DetailRow label="Filepath" value={media.path} fullWidth />
          <DetailRow label="Folder" value={media.folder} fullWidth />
          <DetailRow label="Resolution" value={`${media.width} x ${media.height}`} />
          <DetailRow label="Megapixels" value={megapixels} />
          <DetailRow label="Software" value={media.software} />
          <DetailRow label="UTC offset" value={Math.round(media.offset)} />
          <ColorRow color={media.color} />
          {media.subjects && media.subjects.length > 0 && (
            <div className="-mx-3 flex flex-col border-b border-zinc-300 px-3 py-3 dark:border-zinc-800">
              <span className="mb-1 text-sm font-medium text-zinc-700 dark:text-zinc-500">Tags</span>
              <div className="mt-1 flex flex-wrap gap-2">
                {media.subjects.map((subject) => (
                  <a
                    key={subject.key}
                    href={`/?tag=${encodeURIComponent(subject.key)}`}
                    className="bg-primary-100 text-primary-700 hover:bg-primary-200 dark:bg-primary-900 dark:text-primary-200 dark:hover:bg-primary-800 inline-block rounded px-2 py-1 text-xs font-semibold transition-colors"
                  >
                    {subject.value}
                  </a>
                ))}
              </div>
            </div>
          )}
        </div>
        {/* Camera */}
        <div className="space-y-6">
          <h3 className="mb-6 flex items-center gap-2 border-b border-zinc-300 pb-3 text-xl font-bold text-zinc-900 dark:border-zinc-700 dark:text-zinc-100">
            Camera & Exif
          </h3>

          <div className="group relative overflow-hidden rounded-xl border border-zinc-300 bg-gradient-to-br from-zinc-200 to-zinc-100 p-6 dark:border-zinc-700/50 dark:from-zinc-700 dark:to-zinc-800">
            {media.camera && (
              <div className="relative z-10 mb-4">
                <span className="mb-1 block text-[10px] font-bold tracking-widest text-zinc-500 dark:text-zinc-400">
                  Camera
                </span>
                <div className="text-3xl leading-tight font-bold tracking-tight text-zinc-900 dark:text-white">
                  {media.camera}
                </div>
              </div>
            )}
            {media.lens && (
              <div className="relative z-10 mb-4">
                <span className="mb-1 block text-[10px] font-bold tracking-widest text-zinc-500 dark:text-zinc-400">
                  Lens
                </span>
                <div className="text-xl leading-snug font-medium text-zinc-800 dark:text-zinc-300">{media.lens}</div>
              </div>
            )}
            <div className="relative z-10 border-t border-zinc-300 pt-2 dark:border-zinc-700/50">
              <RatingRow rating={media.rating} />
            </div>
          </div>

          <div className="pt-2">
            <h4 className="mb-4 px-2 text-xs font-semibold tracking-wider text-zinc-500">Exposure Settings</h4>
            <div className="grid grid-cols-2 gap-4">
              <div className="rounded border border-zinc-300 bg-zinc-100/50 p-3 dark:border-zinc-800/50 dark:bg-zinc-950/50">
                <span className="mb-1 block text-xs text-zinc-700 dark:text-zinc-500">Aperture</span>
                <span className="font-mono text-lg text-zinc-900 dark:text-zinc-200">
                  {media.aperture ? `f/${media.aperture}` : '--'}
                </span>
              </div>
              <div className="rounded border border-zinc-300 bg-zinc-100/50 p-3 dark:border-zinc-800/50 dark:bg-zinc-950/50">
                <span className="mb-1 block text-xs text-zinc-500">Shutterspeed</span>
                <span className="font-mono text-lg text-zinc-900 dark:text-zinc-200">{media.shutterspeed || '--'}</span>
              </div>
              <div className="rounded border border-zinc-300 bg-zinc-100/50 p-3 dark:border-zinc-800/50 dark:bg-zinc-950/50">
                <span className="mb-1 block text-xs text-zinc-500">ISO</span>
                <span className="font-mono text-lg text-zinc-900 dark:text-zinc-200">{media.iso || '--'}</span>
              </div>
              <div className="rounded border border-zinc-300 bg-zinc-100/50 p-3 dark:border-zinc-800/50 dark:bg-zinc-950/50">
                <span className="mb-1 block text-xs text-zinc-500">Focus distance</span>
                <span className="font-mono text-lg text-zinc-900 dark:text-zinc-200">
                  {media.focusDistance ? `${Math.round(media.focusDistance).toFixed(2)}m` : '--'}
                </span>
              </div>
              <div className="rounded border border-zinc-300 bg-zinc-100/50 p-3 dark:border-zinc-800/50 dark:bg-zinc-950/50">
                <span className="mb-1 block text-xs text-zinc-500">Focal length</span>
                <span className="font-mono text-lg text-zinc-900 dark:text-zinc-200">
                  {media.focallength ? `${media.focallength}mm` : '--'}
                </span>
              </div>
              <div className="rounded border border-zinc-300 bg-zinc-100/50 p-3 dark:border-zinc-800/50 dark:bg-zinc-950/50">
                <span className="mb-1 block text-xs text-zinc-500">Focal length (35mm)</span>
                <span className="font-mono text-lg text-zinc-900 dark:text-zinc-200">
                  {media.focalLength35 ? `${media.focalLength35}mm` : '--'}
                </span>
              </div>
            </div>
          </div>
        </div>
        {/* Map */}
        <div className="flex flex-col">
          <h3 className="mb-6 flex items-center gap-2 border-b border-zinc-300 pb-3 text-xl font-bold text-zinc-900 dark:border-zinc-700 dark:text-zinc-100">
            Location
          </h3>
          <div className="relative mb-4 h-72 w-full overflow-hidden rounded-xl border border-zinc-300 bg-zinc-100 shadow-inner dark:border-zinc-800 dark:bg-zinc-900">
            {media.latitude && media.longitude ? (
              <MapComponent lat={media.latitude} lng={media.longitude} />
            ) : (
              <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-zinc-900/50 text-zinc-600">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="32"
                  height="32"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0" />
                  <circle cx="12" cy="10" r="3" />
                </svg>
                <span className="text-sm">File does not contain GPS metadata</span>
              </div>
            )}
          </div>
          <div className="space-y-2">
            <DetailRow
              label="Coordinates"
              value={
                media.latitude && media.longitude
                  ? `${media.latitude.toFixed(5)}, ${media.longitude.toFixed(5)}`
                  : undefined
              }
            />
            <DetailRow label="Altitude" value={media.altitude ? `${media.altitude.toFixed(1)}m` : undefined} />
            <DetailRow label="Location" value={media.location} />
          </div>
        </div>
      </div>
    </div>
  );
};

export default ExifPanel;
