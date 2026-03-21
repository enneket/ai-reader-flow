package sqlite

import (
	"ai-rss-reader/internal/models"
)

type NoteRepository struct{}

func NewNoteRepository() *NoteRepository {
	return &NoteRepository{}
}

func (r *NoteRepository) Create(note *models.Note) error {
	result, err := DB.Exec(
		`INSERT INTO notes (article_id, file_path, title, created_at) VALUES (?, ?, ?, ?)`,
		note.ArticleID, note.FilePath, note.Title, note.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	note.ID = id
	return nil
}

func (r *NoteRepository) GetAll() ([]models.Note, error) {
	rows, err := DB.Query(`SELECT id, article_id, file_path, title, created_at FROM notes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var note models.Note
		err := rows.Scan(&note.ID, &note.ArticleID, &note.FilePath, &note.Title, &note.CreatedAt)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}
	return notes, nil
}

func (r *NoteRepository) GetByArticleID(articleID int64) (*models.Note, error) {
	var note models.Note
	err := DB.QueryRow(
		`SELECT id, article_id, file_path, title, created_at FROM notes WHERE article_id = ?`,
		articleID,
	).Scan(&note.ID, &note.ArticleID, &note.FilePath, &note.Title, &note.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

func (r *NoteRepository) DeleteByArticleID(articleID int64) error {
	_, err := DB.Exec(`DELETE FROM notes WHERE article_id = ?`, articleID)
	return err
}
