package repository

import (
	"ai-rss-reader/internal/models"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type NoteFS struct {
	notesDir string
}

func NewNoteFS(notesDir string) *NoteFS {
	return &NoteFS{notesDir: notesDir}
}

func (n *NoteFS) Init() error {
	return os.MkdirAll(n.notesDir, 0755)
}

func (n *NoteFS) CreateNote(article *models.Article, summary string) (string, error) {
	date := time.Now()
	monthDir := filepath.Join(n.notesDir, date.Format("2006-01"))
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		return "", err
	}

	titleSlug := makeSlug(article.Title)
	filename := fmt.Sprintf("%s.md", titleSlug)
	filePath := filepath.Join(monthDir, filename)

	// Handle duplicate filenames
	if _, err := os.Stat(filePath); err == nil {
		filename = fmt.Sprintf("%s-%d.md", titleSlug, date.Unix())
		filePath = filepath.Join(monthDir, filename)
	}

	content := n.formatNote(article, summary)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

func (n *NoteFS) ReadNote(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (n *NoteFS) DeleteNote(filePath string) error {
	return os.Remove(filePath)
}

func (n *NoteFS) ListNotes() ([]string, error) {
	var notes []string
	err := filepath.Walk(n.notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			notes = append(notes, path)
		}
		return nil
	})
	return notes, err
}

func (n *NoteFS) formatNote(article *models.Article, summary string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", article.Title))
	sb.WriteString(fmt.Sprintf("> **Source:** [%s](%s)\n", article.Author, article.Link))
	sb.WriteString(fmt.Sprintf("> **Published:** %s\n\n", article.Published.Format("2006-01-02 15:04")))
	sb.WriteString("---\n\n")
	sb.WriteString("## Summary\n\n")
	sb.WriteString(summary)
	sb.WriteString("\n\n")
	sb.WriteString("---\n\n")
	sb.WriteString("## Original Content\n\n")
	sb.WriteString(article.Content)
	return sb.String()
}

func makeSlug(title string) string {
	// Remove special characters and replace spaces with hyphens
	re := regexp.MustCompile(`[^a-zA-Z0-9\s-]`)
	slug := re.ReplaceAllString(title, "")
	slug = strings.TrimSpace(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ToLower(slug)
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}
