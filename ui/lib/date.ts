export function displayMediaDate(value?: string, offset?: number): string | undefined {
  // https://stackoverflow.com/questions/7403486/add-or-subtract-timezone-difference-to-javascript-date
  if (!value) return undefined;
  const targetTime = new Date(value);
  if (offset) {
    const timeZoneFromDB = offset / 60; //time zone value from database
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
}
