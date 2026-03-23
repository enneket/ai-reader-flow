package sqlite

import (
	"ai-rss-reader/internal/models"
	"database/sql"
	"time"
)

type ArticleRepository struct{}

func NewArticleRepository() *ArticleRepository {
	return &ArticleRepository{}
}

func (r *ArticleRepository) Create(article *models.Article) error {
	status := article.Status
	if status == "" {
		status = "unread"
	}
	result, err := DB.Exec(
		`INSERT INTO articles (feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		article.FeedID, article.Title, article.Link, article.Content, article.Summary,
		article.Author, article.Published.Format(time.RFC3339), article.IsFiltered, article.IsSaved, status, article.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	article.ID = id
	return nil
}

func (r *ArticleRepository) GetByFeedID(feedID int64) ([]models.Article, error) {
	rows, err := DB.Query(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at
		FROM articles WHERE feed_id = ? ORDER BY published DESC`,
		feedID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

func (r *ArticleRepository) GetAll(filterMode string) ([]models.Article, error) {
	query := `SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at FROM articles`
	switch filterMode {
	case "filtered":
		query += ` WHERE is_filtered = 1`
	case "saved":
		query += ` WHERE is_saved = 1`
	case "unread":
		query += ` WHERE status = 'unread'`
	case "accepted":
		query += ` WHERE status = 'accepted'`
	case "rejected":
		query += ` WHERE status = 'rejected'`
	case "snoozed":
		query += ` WHERE status = 'snoozed'`
	}
	query += ` ORDER BY published DESC`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

func (r *ArticleRepository) GetFiltered() ([]models.Article, error) {
	return r.GetAll("filtered")
}

func (r *ArticleRepository) GetSaved() ([]models.Article, error) {
	return r.GetAll("saved")
}

func (r *ArticleRepository) GetByID(id int64) (*models.Article, error) {
	row := DB.QueryRow(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at
		FROM articles WHERE id = ?`,
		id,
	)

	var a models.Article
	var published, createdAt sql.NullString
	err := row.Scan(&a.ID, &a.FeedID, &a.Title, &a.Link, &a.Content, &a.Summary, &a.Author, &published, &a.IsFiltered, &a.IsSaved, &a.Status, &createdAt)
	if err != nil {
		return nil, err
	}
	if published.Valid {
		a.Published, _ = time.Parse(time.RFC3339, published.String)
	}
	if createdAt.Valid {
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	return &a, nil
}

func (r *ArticleRepository) Update(article *models.Article) error {
	_, err := DB.Exec(
		`UPDATE articles SET title = ?, content = ?, summary = ?, is_filtered = ?, is_saved = ?, status = ? WHERE id = ?`,
		article.Title, article.Content, article.Summary, article.IsFiltered, article.IsSaved, article.Status, article.ID,
	)
	return err
}

func (r *ArticleRepository) SetFiltered(id int64, filtered bool) error {
	_, err := DB.Exec(`UPDATE articles SET is_filtered = ? WHERE id = ?`, filtered, id)
	return err
}

func (r *ArticleRepository) SetSaved(id int64, saved bool) error {
	_, err := DB.Exec(`UPDATE articles SET is_saved = ? WHERE id = ?`, saved, id)
	return err
}

func (r *ArticleRepository) SetStatus(id int64, status string) error {
	_, err := DB.Exec(`UPDATE articles SET status = ? WHERE id = ?`, status, id)
	return err
}

func (r *ArticleRepository) GetByStatus(status string) ([]models.Article, error) {
	return r.GetAll(status)
}

func (r *ArticleRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM articles WHERE id = ?`, id)
	return err
}

func (r *ArticleRepository) LinkExists(link string) (bool, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM articles WHERE link = ?`, link).Scan(&count)
	return count > 0, err
}

func (r *ArticleRepository) scanArticles(rows *sql.Rows) ([]models.Article, error) {
	var articles []models.Article
	for rows.Next() {
		var a models.Article
		var published, createdAt sql.NullString
		err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.Link, &a.Content, &a.Summary, &a.Author, &published, &a.IsFiltered, &a.IsSaved, &a.Status, &createdAt)
		if err != nil {
			continue
		}
		if published.Valid {
			a.Published, _ = time.Parse(time.RFC3339, published.String)
		}
		if createdAt.Valid {
			a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}
		if a.Status == "" {
			a.Status = "unread"
		}
		articles = append(articles, a)
	}
	return articles, nil
}
