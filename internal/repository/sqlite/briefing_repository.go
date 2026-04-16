package sqlite

import (
	"ai-rss-reader/internal/models"
	"database/sql"
	"log"
	"time"
)

type BriefingRepository struct{}

func NewBriefingRepository() *BriefingRepository {
	return &BriefingRepository{}
}

func (r *BriefingRepository) Create(b *models.Briefing) error {
	result, err := DB.Exec(
		`INSERT INTO briefings (status, created_at) VALUES (?, ?)`,
		b.Status, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	b.ID = id
	return nil
}

func (r *BriefingRepository) GetByID(id int64) (*models.Briefing, error) {
	row := DB.QueryRow(
		`SELECT id, status, COALESCE(title, ''), COALESCE(lead, ''), COALESCE(closing, ''), error, created_at, completed_at FROM briefings WHERE id = ?`,
		id,
	)
	var b models.Briefing
	var createdAt, completedAt string
	err := row.Scan(&b.ID, &b.Status, &b.Title, &b.Lead, &b.Closing, &b.Error, &createdAt, &completedAt)
	if err != nil {
		return nil, err
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if completedAt != "" {
		t, _ := time.Parse(time.RFC3339, completedAt)
		b.CompletedAt = &t
	}
	return &b, nil
}

func (r *BriefingRepository) GetAll(limit, offset int) ([]models.Briefing, error) {
	rows, err := DB.Query(
		`SELECT id, status, COALESCE(title, ''), COALESCE(lead, ''), COALESCE(closing, ''), error, created_at, completed_at FROM briefings ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var briefings []models.Briefing
	for rows.Next() {
		var b models.Briefing
		var createdAt, completedAt string
		err := rows.Scan(&b.ID, &b.Status, &b.Title, &b.Lead, &b.Closing, &b.Error, &createdAt, &completedAt)
		if err != nil {
			continue
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if completedAt != "" {
			t, _ := time.Parse(time.RFC3339, completedAt)
			b.CompletedAt = &t
		}
		briefings = append(briefings, b)
	}
	return briefings, nil
}

func (r *BriefingRepository) UpdateStatus(id int64, status string, errMsg string) error {
	var completedAt string
	if status == "completed" || status == "failed" {
		completedAt = time.Now().Format(time.RFC3339)
	}
	_, err := DB.Exec(
		`UPDATE briefings SET status = ?, error = ?, completed_at = ? WHERE id = ?`,
		status, errMsg, completedAt, id,
	)
	return err
}

func (r *BriefingRepository) UpdateBriefingMeta(id int64, title, lead, closing string) error {
	_, err := DB.Exec(
		`UPDATE briefings SET title = ?, lead = ?, closing = ? WHERE id = ?`,
		title, lead, closing, id,
	)
	return err
}

func (r *BriefingRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM briefings WHERE id = ?`, id)
	return err
}

func (r *BriefingRepository) DeleteAll() error {
	_, err := DB.Exec(`DELETE FROM briefing_items`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM briefings`)
	return err
}

func (r *BriefingRepository) CreateItem(item *models.BriefingItem) error {
	result, err := DB.Exec(
		`INSERT INTO briefing_items (briefing_id, topic, summary, sort_order) VALUES (?, ?, ?, ?)`,
		item.BriefingID, item.Topic, item.Summary, item.SortOrder,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

func (r *BriefingRepository) GetItemsByBriefingID(briefingID int64) ([]models.BriefingItem, error) {
	rows, err := DB.Query(
		`SELECT id, briefing_id, topic, summary, sort_order FROM briefing_items WHERE briefing_id = ? ORDER BY sort_order`,
		briefingID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.BriefingItem
	for rows.Next() {
		var item models.BriefingItem
		err := rows.Scan(&item.ID, &item.BriefingID, &item.Topic, &item.Summary, &item.SortOrder)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *BriefingRepository) CreateArticle(article *models.BriefingArticle) error {
	result, err := DB.Exec(
		`INSERT INTO briefing_articles (briefing_item_id, article_id, title, insight, key_argument, source_url) VALUES (?, ?, ?, ?, ?, ?)`,
		article.BriefingItemID, article.ArticleID, article.Title, article.Insight, article.KeyArgument, article.SourceURL,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	article.ID = id
	return nil
}

func (r *BriefingRepository) GetArticlesByItemID(itemID int64) ([]models.BriefingArticle, error) {
	rows, err := DB.Query(
		`SELECT id, briefing_item_id, article_id, title, COALESCE(insight, ''), COALESCE(key_argument, ''), COALESCE(source_url, '') FROM briefing_articles WHERE briefing_item_id = ?`,
		itemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.BriefingArticle
	for rows.Next() {
		var a models.BriefingArticle
		err := rows.Scan(&a.ID, &a.BriefingItemID, &a.ArticleID, &a.Title, &a.Insight, &a.KeyArgument, &a.SourceURL)
		if err != nil {
			continue
		}
		articles = append(articles, a)
	}
	return articles, nil
}

// GetLatestBriefingTime returns the created_at of the most recent completed briefing
func (r *BriefingRepository) GetLatestBriefingTime() (*time.Time, error) {
	row := DB.QueryRow(
		`SELECT created_at FROM briefings WHERE status = 'completed' ORDER BY created_at DESC LIMIT 1`,
	)
	var createdAt string
	err := row.Scan(&createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339, createdAt)
	return &t, nil
}

// TryAcquireCronLock atomically acquires a cron lock for the given time slot (e.g. "09:00").
// Returns true if acquired, false if another cron instance already holds the lock.
func (r *BriefingRepository) TryAcquireCronLock(timeSlot string) bool {
	now := time.Now()
	expiresAt := now.Add(65 * time.Minute).Format(time.RFC3339)
	nowStr := now.Format(time.RFC3339)
	// INSERT OR IGNORE: if the lock already exists and hasn't expired, the insert is ignored (no row affected)
	result, err := DB.Exec(
		`INSERT OR IGNORE INTO cron_locks (time_slot, locked_at, expires_at) VALUES (?, ?, ?)`,
		timeSlot, nowStr, expiresAt,
	)
	if err != nil {
		log.Printf("[cron] lock insert error: %v", err)
		return false
	}
	affected, _ := result.RowsAffected()
	if affected == 1 {
		return true // lock acquired
	}
	// Lock already exists — check if expired
	row := DB.QueryRow(`SELECT expires_at FROM cron_locks WHERE time_slot = ?`, timeSlot)
	var expiresAtStr string
	if err := row.Scan(&expiresAtStr); err != nil {
		return false
	}
	expiresAtParsed, _ := time.Parse(time.RFC3339, expiresAtStr)
	if now.After(expiresAtParsed) {
		// Expired — delete and retry
		DB.Exec(`DELETE FROM cron_locks WHERE time_slot = ?`, timeSlot)
		result2, err2 := DB.Exec(
			`INSERT OR IGNORE INTO cron_locks (time_slot, locked_at, expires_at) VALUES (?, ?, ?)`,
			timeSlot, nowStr, expiresAt,
		)
		if err2 != nil {
			return false
		}
		affected2, _ := result2.RowsAffected()
		return affected2 == 1
	}
	return false // lock held by another instance
}

// ReleaseCronLock releases the cron lock for the given time slot.
func (r *BriefingRepository) ReleaseCronLock(timeSlot string) {
	DB.Exec(`DELETE FROM cron_locks WHERE time_slot = ?`, timeSlot)
}
