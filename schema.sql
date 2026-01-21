-- SQLite schema for notes converter database
-- Table: notes

CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date_created DATETIME DEFAULT CURRENT_TIMESTAMP,
    image TEXT NOT NULL,
    markdown TEXT NOT NULL
);

-- Index for faster lookups by creation date
CREATE INDEX IF NOT EXISTS idx_notes_date_created ON notes(date_created);

-- Index for image file path lookups
CREATE INDEX IF NOT EXISTS idx_notes_image ON notes(image);