package sqlite

import (
	"ai-rss-reader/internal/models"
	"database/sql"
	"fmt"
	"log"
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
		`INSERT INTO articles (feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		article.FeedID, article.Title, article.Link, article.Content, article.Summary,
		article.Author, article.Published.Format(time.RFC3339), article.IsFiltered, article.IsSaved, status, article.CreatedAt.Format(time.RFC3339), article.IsTranslated, article.TranslatedContent,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	article.ID = id
	// Index in FTS for full-text search
	_ = IndexArticle(article.ID, article.Title, article.Content)
	// 同步更新 feed 的 unread_count
	if status == "unread" {
		DB.Exec(`UPDATE feeds SET unread_count = unread_count + 1 WHERE id = ?`, article.FeedID)
	}
	return nil
}

func (r *ArticleRepository) GetByFeedID(feedID int64, limit, offset int) ([]models.Article, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := DB.Query(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles WHERE feed_id = ? ORDER BY published DESC LIMIT ? OFFSET ?`,
		feedID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

func (r *ArticleRepository) GetAll(filterMode string, limit, offset int) ([]models.Article, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles`
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
	query += ` ORDER BY published DESC LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

func (r *ArticleRepository) GetFiltered() ([]models.Article, error) {
	return r.GetAll("filtered", 0, 0)
}

func (r *ArticleRepository) GetSaved() ([]models.Article, error) {
	return r.GetAll("saved", 0, 0)
}

func (r *ArticleRepository) GetByID(id int64) (*models.Article, error) {
	row := DB.QueryRow(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles WHERE id = ?`,
		id,
	)

	var a models.Article
	var published, createdAt sql.NullString
	var isTranslated sql.NullBool
	var translatedContent sql.NullString
	err := row.Scan(&a.ID, &a.FeedID, &a.Title, &a.Link, &a.Content, &a.Summary, &a.Author, &published, &a.IsFiltered, &a.IsSaved, &a.Status, &createdAt, &isTranslated, &translatedContent)
	if isTranslated.Valid {
		a.IsTranslated = isTranslated.Bool
	}
	if translatedContent.Valid {
		a.TranslatedContent = translatedContent.String
	}
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

func (r *ArticleRepository) GetByIDs(ids []int64) ([]models.Article, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// Build placeholders: (?, ?, ?)
	placeholders := ""
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}
	query := fmt.Sprintf(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles WHERE id IN (%s)`, placeholders)
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanArticles(rows)
}

func (r *ArticleRepository) Update(article *models.Article) error {
	_, err := DB.Exec(
		`UPDATE articles SET title = ?, content = ?, summary = ?, is_filtered = ?, is_saved = ?, status = ?, is_translated = ?, translated_content = ? WHERE id = ?`,
		article.Title, article.Content, article.Summary, article.IsFiltered, article.IsSaved, article.Status, article.IsTranslated, article.TranslatedContent, article.ID,
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
	// 获取旧 status 和 feed_id，用于更新 unread_count
	var oldStatus string
	var feedId int64
	err := DB.QueryRow(`SELECT status, feed_id FROM articles WHERE id = ?`, id).Scan(&oldStatus, &feedId)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`UPDATE articles SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}

	// 同步更新 feed 的 unread_count
	if oldStatus == "unread" && status != "unread" {
		DB.Exec(`UPDATE feeds SET unread_count = MAX(0, unread_count - 1) WHERE id = ?`, feedId)
	} else if oldStatus != "unread" && status == "unread" {
		DB.Exec(`UPDATE feeds SET unread_count = unread_count + 1 WHERE id = ?`, feedId)
	}

	return nil
}

func (r *ArticleRepository) GetByStatus(status string) ([]models.Article, error) {
	return r.GetAll(status, 0, 0)
}

func (r *ArticleRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_ = RemoveArticleFTS(id)
	return nil
}

// GetRecentForBriefing returns recent unread articles
func (r *ArticleRepository) GetRecentForBriefing() ([]models.Article, error) {
	rows, err := DB.Query(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles
         WHERE status = 'unread'
         ORDER BY created_at DESC
         LIMIT 100`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanArticles(rows)
}

// GetArticlesAfter returns articles created after the given time
func (r *ArticleRepository) GetArticlesAfter(startTime time.Time) ([]models.Article, error) {
	rows, err := DB.Query(
		`SELECT id, feed_id, title, link, content, summary, author, published, is_filtered, is_saved, status, created_at, is_translated, translated_content FROM articles
         WHERE created_at > ?
         ORDER BY created_at DESC
         LIMIT 100`,
		startTime,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanArticles(rows)
}

func (r *ArticleRepository) LinkExists(link string) (bool, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM articles WHERE link = ?`, link).Scan(&count)
	return count > 0, err
}

func (r *ArticleRepository) Search(query string, limit int) ([]models.Article, error) {
	if limit <= 0 {
		limit = 20
	}
	// FTS5 search on title + content, return matching article IDs
	rows, err := DB.Query(`
		SELECT a.id, a.feed_id, a.title, a.link, a.content, a.summary, a.author, a.published,
		       a.is_filtered, a.is_saved, a.status, a.created_at, a.is_translated, a.translated_content
		FROM articles a
		JOIN articles_fts fts ON a.id = fts.article_id
		WHERE articles_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanArticles(rows)
}

func (r *ArticleRepository) scanArticles(rows *sql.Rows) ([]models.Article, error) {
	var articles = []models.Article{}
	for rows.Next() {
		var a models.Article
		var published, createdAt sql.NullString
		var isTranslated sql.NullBool
		var translatedContent sql.NullString
		err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.Link, &a.Content, &a.Summary, &a.Author, &published, &a.IsFiltered, &a.IsSaved, &a.Status, &createdAt, &isTranslated, &translatedContent)
		if isTranslated.Valid {
			a.IsTranslated = isTranslated.Bool
		}
		if translatedContent.Valid {
			a.TranslatedContent = translatedContent.String
		}
		if err != nil {
			log.Printf("scan article error (row may be skipped): %v", err)
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
