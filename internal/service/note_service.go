package service

import (
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository"
	"ai-rss-reader/internal/repository/sqlite"
	"fmt"
	"time"
)

type NoteService struct {
	noteRepo *sqlite.NoteRepository
	noteFS   *repository.NoteFS
}

func NewNoteService(notesDir string) *NoteService {
	return &NoteService{
		noteRepo: sqlite.NewNoteRepository(),
		noteFS:   repository.NewNoteFS(notesDir),
	}
}

func (s *NoteService) Init() error {
	return s.noteFS.Init()
}

func (s *NoteService) CreateNote(article *models.Article, summary string) (*models.Note, error) {
	// Check if note already exists
	existing, _ := s.noteRepo.GetByArticleID(article.ID)
	if existing != nil {
		return existing, nil
	}

	// Create markdown file
	filePath, err := s.noteFS.CreateNote(article, summary)
	if err != nil {
		return nil, fmt.Errorf("failed to create note file: %w", err)
	}

	// Save note index
	note := &models.Note{
		ArticleID: article.ID,
		FilePath:  filePath,
		Title:     article.Title,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	if err := s.noteRepo.Create(note); err != nil {
		// Try to cleanup the file
		s.noteFS.DeleteNote(filePath)
		return nil, fmt.Errorf("failed to save note index: %w", err)
	}

	// Mark article as saved
	articleRepo := sqlite.NewArticleRepository()
	articleRepo.SetSaved(article.ID, true)

	return note, nil
}

func (s *NoteService) GetNotes() ([]models.Note, error) {
	return s.noteRepo.GetAll()
}

func (s *NoteService) GetNoteByArticleID(articleID int64) (*models.Note, error) {
	return s.noteRepo.GetByArticleID(articleID)
}

func (s *NoteService) ReadNote(note *models.Note) (string, error) {
	return s.noteFS.ReadNote(note.FilePath)
}

func (s *NoteService) DeleteNote(noteID int64) error {
	note, err := s.noteRepo.GetByArticleID(noteID)
	if err != nil {
		return err
	}

	// Delete file
	if err := s.noteFS.DeleteNote(note.FilePath); err != nil {
		fmt.Printf("Warning: failed to delete note file: %v\n", err)
	}

	// Delete index
	if err := s.noteRepo.Delete(note.ID); err != nil {
		return err
	}

	// Mark article as unsaved
	articleRepo := sqlite.NewArticleRepository()
	articleRepo.SetSaved(note.ArticleID, false)

	return nil
}
