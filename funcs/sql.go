package funcs

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Note represents a note entry in the database
type Note struct {
	ID          int       `json:"id"`
	DateCreated time.Time `json:"date_created"`
	Image       string    `json:"image"`
	Markdown    string    `json:"markdown"`
}

// AddNote inserts a new note into the database
func AddNote(db *sql.DB, image, markdown string) (*Note, error) {
	query := `INSERT INTO notes (image, markdown) VALUES (?, ?)`
	result, err := db.Exec(query, image, markdown)
	if err != nil {
		return nil, fmt.Errorf("failed to insert note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Retrieve the newly created note
	note := &Note{
		ID:          int(id),
		DateCreated: time.Now(),
		Image:       image,
		Markdown:    markdown,
	}

	return note, nil
}

// UpdateNote updates an existing note in the database
func UpdateNote(db *sql.DB, id int, image, markdown string) (*Note, error) {
	query := `UPDATE notes SET image = ?, markdown = ? WHERE id = ?`
	result, err := db.Exec(query, image, markdown, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no note found with id %d", id)
	}

	// Retrieve the updated note
	return GetNoteByID(db, id)
}

// DeleteNote removes a note from the database by ID
func DeleteNote(db *sql.DB, id int) error {
	query := `DELETE FROM notes WHERE id = ?`
	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no note found with id %d", id)
	}

	return nil
}

// GetNoteByID retrieves a note by its ID
func GetNoteByID(db *sql.DB, id int) (*Note, error) {
	query := `SELECT id, date_created, image, markdown FROM notes WHERE id = ?`
	row := db.QueryRow(query, id)

	var note Note
	err := row.Scan(&note.ID, &note.DateCreated, &note.Image, &note.Markdown)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no note found with id %d", id)
		}
		return nil, fmt.Errorf("failed to scan note: %w", err)
	}

	return &note, nil
}

// GetAllNotes retrieves all notes from the database
func GetAllNotes(db *sql.DB) ([]Note, error) {
	query := `SELECT id, date_created, image, markdown FROM notes ORDER BY date_created DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(&note.ID, &note.DateCreated, &note.Image, &note.Markdown)
		if err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// InitDB initializes a new SQLite database connection and creates the schema
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date_created DATETIME DEFAULT CURRENT_TIMESTAMP,
		image TEXT NOT NULL,
		markdown TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_notes_date_created ON notes(date_created);
	CREATE INDEX IF NOT EXISTS idx_notes_image ON notes(image);
	`

	if _, err = db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}
