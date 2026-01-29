PRAGMA journal_mode = WAL;

PRAGMA temp_store = 2;

CREATE TABLE
  IF NOT EXISTS media (
    "hash" INTEGER NOT NULL PRIMARY KEY,
    "path" TEXT,
    "subject" TEXT,
    "width" INTEGER,
    "height" INTEGER,
    "ratio" REAL,
    "padding" INTEGER,
    "date" TEXT,
    "modified" TEXT,
    "folder" TEXT,
    "rating" REAL,
    shutterspeed TEXT DEFAULT '',
    aperture REAL DEFAULT '',
    iso REAL DEFAULT '',
    lens TEXT DEFAULT '',
    camera TEXT DEFAULT '',
    focallength REAL DEFAULT '',
    altitude REAL DEFAULT '',
    latitude REAL DEFAULT '',
    longitude REAL DEFAULT '',
    mediatype TEXT DEFAULT '',
    focusdistance REAL DEFAULT '',
    focallength35 REAL DEFAULT '',
    color TEXT DEFAULT '',
    location TEXT DEFAULT '',
    description TEXT DEFAULT '',
    title TEXT DEFAULT '',
    software TEXT DEFAULT '',
    offset
      REAL DEFAULT 0,
      rotation REAL DEFAULT 0,
      "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      UNIQUE (hash)
  );

CREATE TABLE
  IF NOT EXISTS schema (
    "key" TEXT PRIMARY KEY DEFAULT 'version',
    "value" INTEGER DEFAULT '0',
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (key)
  );

CREATE TABLE
  IF NOT EXISTS tags (
    "id" INTEGER NOT NULL PRIMARY KEY,
    "key" TEXT,
    "value" TEXT,
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (id)
  );

CREATE TABLE
  IF NOT EXISTS folders (
    "id" INTEGER NOT NULL PRIMARY KEY,
    "key" TEXT,
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (key) REFERENCES media (folder),
    UNIQUE (key)
  );

CREATE TABLE
  IF NOT EXISTS images_tags (
    "id" INTEGER NOT NULL PRIMARY KEY,
    "image_id" INTEGER,
    "tag_id" INTEGER,
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (image_id) REFERENCES media (hash),
    FOREIGN KEY (tag_id) REFERENCES tags (id),
    UNIQUE (id)
  );

CREATE TABLE
  IF NOT EXISTS users (
    "username" TEXT NOT NULL PRIMARY KEY,
    "password" TEXT,
    "role" TEXT,
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (username)
  );

DROP TABLE IF EXISTS images_virtual;

CREATE VIRTUAL TABLE IF NOT EXISTS images_virtual USING FTS5 (
  hash,
  path,
  subject,
  width,
  height,
  ratio,
  padding,
  date,
  modified,
  folder,
  rating,
  shutterspeed,
  aperture,
  iso,
  lens,
  camera,
  focallength,
  altitude,
  latitude,
  longitude,
  mediatype,
  focusdistance,
  focallength35,
  color,
  location,
  description,
  title,
  software,
  offset
,
    rotation,
    created,
    tokenize = 'trigram'
);

INSERT INTO
  images_virtual
SELECT
  *
FROM
  media;

DROP TRIGGER IF EXISTS images_insert;

CREATE TRIGGER IF NOT EXISTS images_insert AFTER INSERT ON media BEGIN
INSERT INTO
  images_virtual (
    rowid,
    hash,
    path,
    subject,
    width,
    height,
    ratio,
    padding,
    date,
    modified,
    folder,
    rating,
    shutterspeed,
    aperture,
    iso,
    lens,
    camera,
    focallength,
    altitude,
    latitude,
    longitude,
    mediatype,
    focusdistance,
    focallength35,
    color,
    location,
    description,
    title,
    software,
    offset
,
      rotation,
      created
  )
VALUES
  (
    new.ROWID,
    new.hash,
    new.path,
    new.subject,
    new.width,
    new.height,
    new.ratio,
    new.padding,
    new.date,
    new.modified,
    new.folder,
    new.rating,
    new.shutterspeed,
    new.aperture,
    new.iso,
    new.lens,
    new.camera,
    new.focallength,
    new.altitude,
    new.latitude,
    new.longitude,
    new.mediatype,
    new.focusdistance,
    new.focallength35,
    new.color,
    new.location,
    new.description,
    new.title,
    new.software,
    new.offset,
    new.rotation,
    new.created
  );

END;

DROP TRIGGER IF EXISTS images_delete;

CREATE TRIGGER IF NOT EXISTS images_delete AFTER DELETE ON media BEGIN
DELETE FROM images_virtual
WHERE
  hash = OLD.hash;

END;

CREATE TABLE
  IF NOT EXISTS keys (
    "name" TEXT NOT NULL PRIMARY KEY,
    "key" TEXT,
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name)
  );

CREATE TABLE
  IF NOT EXISTS scan_errors (
    "path" TEXT,
    "modified" TEXT,
    "error" TEXT,
    "created" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (path)
  );

CREATE INDEX IF NOT EXISTS idx_media_folder_date ON media (folder, date DESC);

CREATE INDEX IF NOT EXISTS idx_media_folder ON media (folder);

CREATE INDEX IF NOT EXISTS idx_folders_key ON folders (key);

CREATE TABLE
  IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

CREATE TABLE
  IF NOT EXISTS notifications_dismissed (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    notification_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    dismissed BOOLEAN DEFAULT 0,
    dismissed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (notification_id) REFERENCES notifications (id),
    FOREIGN KEY (username) REFERENCES users (username),
    UNIQUE (notification_id, username)
  );
