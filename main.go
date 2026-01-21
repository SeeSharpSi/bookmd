package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"seesharpsi/bookmd/funcs"
	"seesharpsi/bookmd/templ"

	_ "modernc.org/sqlite"
)

var (
	db       *sql.DB
	aiClient *openai.Client
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	port := flag.Int("port", 9779, "port the server runs on")
	address := flag.String("address", "http://localhost", "address the server runs on")
	flag.Parse()

	// Initialize database
	var err error
	db, err = funcs.InitDB("./notes.db")
	if err != nil {
		log.Panic("failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize OpenAI client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set, AI features will not work")
	} else {
		config := openai.DefaultConfig(apiKey)
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
		aiClient = openai.NewClientWithConfig(config)
	}

	// Create images directory if it doesn't exist
	if err := os.MkdirAll("./images", 0755); err != nil {
		log.Panic("failed to create images directory:", err)
	}

	// ip parsing
	base_ip := *address
	ip := base_ip + ":" + strconv.Itoa(*port)
	root_ip, err := url.Parse(ip)
	if err != nil {
		log.Panic(err)
	}

	mux := http.NewServeMux()
	add_routes(mux)

	server := http.Server{
		Addr:    root_ip.Host,
		Handler: mux,
	}

	// start server
	log.Printf("running server on %s\n", root_ip.Host)
	err = server.ListenAndServe()
	defer server.Close()
	if errors.Is(err, http.ErrServerClosed) {
		log.Printf("server closed\n")
	} else if err != nil {
		log.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func add_routes(mux *http.ServeMux) {
	mux.HandleFunc("/", GetIndex)
	mux.HandleFunc("/static/{file}", ServeStatic)
	mux.HandleFunc("/api/add-note", AddNoteHandler)
	mux.HandleFunc("/api/update-note", UpdateNoteHandler)
	mux.HandleFunc("/api/regenerate-note", RegenerateNoteHandler)
}

func ServeStatic(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	log.Printf("got /static/%s request\n", file)
	http.ServeFile(w, r, "./static/"+file)
}

func GetIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("got / request\n")
	component := templ.Index()
	component.Render(context.Background(), w)
}

func AddNoteHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("got %s request\n", r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "No image file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d%s", header.Size, ext)
	imagePath := filepath.Join("./images", filename)

	// Save image to images folder
	dst, err := os.Create(imagePath)
	if err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	// Convert image to markdown using AI
	markdown, err := funcs.ConvertImageToMarkdown(context.Background(), aiClient, imagePath)
	if err != nil {
		print(err.Error())
		http.Error(w, "Failed to convert image to markdown", http.StatusInternalServerError)
		return
	}

	// Save to database
	note, err := funcs.AddNote(db, filename, markdown)
	if err != nil {
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "id": %d, "image": "%s", "markdown": "%s"}`,
		note.ID, note.Image, strings.ReplaceAll(note.Markdown, "\n", "\\n"))
}

func UpdateNoteHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("got %s request\n", r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get note ID from form
	idStr := r.FormValue("id")
	if idStr == "" {
		http.Error(w, "Note ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "No image file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d%s", header.Size, ext)
	imagePath := filepath.Join("./images", filename)

	// Save image to images folder
	dst, err := os.Create(imagePath)
	if err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	// Convert image to markdown using AI
	markdown, err := funcs.ConvertImageToMarkdown(context.Background(), aiClient, imagePath)
	if err != nil {
		http.Error(w, "Failed to convert image to markdown", http.StatusInternalServerError)
		return
	}

	// Update database
	note, err := funcs.UpdateNote(db, id, filename, markdown)
	if err != nil {
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "id": %d, "image": "%s", "markdown": "%s"}`,
		note.ID, note.Image, strings.ReplaceAll(note.Markdown, "\n", "\\n"))
}

func RegenerateNoteHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("got %s request\n", r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get note ID from form
	idStr := r.FormValue("id")
	if idStr == "" {
		http.Error(w, "Note ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the existing note from database
	note, err := funcs.GetNoteByID(db, id)
	if err != nil {
		http.Error(w, "Failed to retrieve note: "+err.Error(), http.StatusNotFound)
		return
	}

	// Construct full image path
	imagePath := filepath.Join("./images", note.Image)

	// Check if image file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		http.Error(w, "Image file not found", http.StatusNotFound)
		return
	}

	// Convert image to markdown using AI (regenerating)
	markdown, err := funcs.ConvertImageToMarkdown(context.Background(), aiClient, imagePath)
	if err != nil {
		http.Error(w, "Failed to convert image to markdown: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update database with new markdown (keeping same image)
	updatedNote, err := funcs.UpdateNote(db, id, note.Image, markdown)
	if err != nil {
		http.Error(w, "Failed to update database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "id": %d, "image": "%s", "markdown": "%s"}`,
		updatedNote.ID, updatedNote.Image, strings.ReplaceAll(updatedNote.Markdown, "\n", "\\n"))
}
