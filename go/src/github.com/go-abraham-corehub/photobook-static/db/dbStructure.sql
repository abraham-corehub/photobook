CREATE TABLE image (
  id INTEGER PRIMARY KEY AUTOINCREMENT UNIQUE NOT NULL,
  location TEXT NOT NULL,
  id_album INTEGER,
  id_user INTEGER
);