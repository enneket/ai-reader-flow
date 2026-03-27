package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-rss-reader/internal/models"
)

func TestNoteFSInit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "note_fs_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := NewNoteFS(tmpDir)
	err = n.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat notes dir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected directory, got %v", info.Mode())
	}
}

func TestNoteFSCreateNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "note_fs_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := NewNoteFS(tmpDir)
	if err := n.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	article := &models.Article{
		Title:    "Test Article Title",
		Link:     "https://example.com/article",
		Author:   "Test Author",
		Content:  "This is the article content.",
		Published: time.Now(),
	}

	path, err := n.CreateNote(article, "This is a test summary.")
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("CreateNote() created file but file does not exist at %s", path)
	}

	// Verify it's a .md file
	if !strings.HasSuffix(path, ".md") {
		t.Errorf("CreateNote() path = %s, expected .md suffix", path)
	}
}

func TestNoteFSReadNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "note_fs_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := NewNoteFS(tmpDir)
	if err := n.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	article := &models.Article{
		Title:    "Read Test Article",
		Link:     "https://example.com/read-test",
		Author:   "Read Author",
		Content:  "This content should be readable.",
		Published: time.Now(),
	}

	path, err := n.CreateNote(article, "Summary for reading.")
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}

	content, err := n.ReadNote(path)
	if err != nil {
		t.Fatalf("ReadNote() error = %v", err)
	}

	if !strings.Contains(content, "Read Test Article") {
		t.Errorf("ReadNote() content missing title")
	}
	if !strings.Contains(content, "Summary for reading") {
		t.Errorf("ReadNote() content missing summary")
	}
}

func TestNoteFSDeleteNote(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "note_fs_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := NewNoteFS(tmpDir)
	if err := n.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	article := &models.Article{
		Title:    "Delete Test Article",
		Link:     "https://example.com/delete-test",
		Author:   "Delete Author",
		Content:  "This content will be deleted.",
		Published: time.Now(),
	}

	path, err := n.CreateNote(article, "Summary.")
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}

	err = n.DeleteNote(path)
	if err != nil {
		t.Fatalf("DeleteNote() error = %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("DeleteNote() file still exists at %s", path)
	}
}

func TestNoteFSListNotes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "note_fs_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := NewNoteFS(tmpDir)
	if err := n.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Initially no notes
	notes, err := n.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("ListNotes() = %d notes, want 0", len(notes))
	}

	// Create some notes
	for i := 0; i < 3; i++ {
		article := &models.Article{
			Title:    filepath.Join("Test Note", string(rune('A'+i))),
			Link:     "https://example.com/note",
			Author:   "Author",
			Content:  "Content",
			Published: time.Now(),
		}
		_, err := n.CreateNote(article, "Summary")
		if err != nil {
			t.Fatalf("CreateNote() error = %v", err)
		}
	}

	notes, err = n.ListNotes()
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 3 {
		t.Errorf("ListNotes() = %d notes, want 3", len(notes))
	}

	// All should be .md files
	for _, note := range notes {
		if !strings.HasSuffix(note, ".md") {
			t.Errorf("ListNotes() returned non-.md file: %s", note)
		}
	}
}

func TestMakeSlug(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Simple Title", "simple-title"},
		{"Title With Special!@#$Chars", "title-with-specialchars"},
		{"  Leading Trailing  ", "leading-trailing"},
		{"UPPERCASE Title", "uppercase-title"},
		{"Very Long Title That Exceeds Fifty Characters Maximum", "very-long-title-that-exceeds-fifty-characters-maxi"},
		{"Title   Multiple   Spaces", "title---multiple---spaces"},
		{"Title-With-Dashes", "title-with-dashes"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := makeSlug(tt.title)
			if result != tt.expected {
				t.Errorf("makeSlug(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}
