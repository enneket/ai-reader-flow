package sqlite

import (
	"ai-rss-reader/internal/models"
	"database/sql"
	"log"
	"time"
)

type FeedRepository struct{}

func NewFeedRepository() *FeedRepository {
	return &FeedRepository{}
}

func (r *FeedRepository) Create(feed *models.Feed) error {
	result, err := DB.Exec(
		`INSERT INTO feeds (title, url, description, icon_url, last_fetched, is_dead, created_at, group_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		feed.Title, feed.URL, feed.Description, feed.IconURL, feed.LastFetched.Format(time.RFC3339), false, feed.CreatedAt.Format(time.RFC3339), feed.Group,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	feed.ID = id
	return nil
}

func (r *FeedRepository) GetAll() ([]models.Feed, error) {
	rows, err := DB.Query(`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at, COALESCE(group_name, ''),
	        last_refresh_success, COALESCE(last_refresh_error, ''), last_refreshed, unread_count
	 FROM feeds ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds = []models.Feed{}
	for rows.Next() {
		var f models.Feed
		var lastFetched, createdAt, lastRefreshed sql.NullString
		err := rows.Scan(&f.ID, &f.Title, &f.URL, &f.Description, &f.IconURL, &lastFetched, &f.IsDead, &createdAt, &f.Group,
			&f.LastRefreshSuccess, &f.LastRefreshError, &lastRefreshed, &f.UnreadCount)
		if err != nil {
			log.Printf("scan feed error (row may be skipped): %v", err)
			continue
		}
		if lastFetched.Valid {
			f.LastFetched, _ = time.Parse(time.RFC3339, lastFetched.String)
		}
		if createdAt.Valid {
			f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}
		if lastRefreshed.Valid {
			f.LastRefreshed, _ = time.Parse(time.RFC3339, lastRefreshed.String)
		}
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (r *FeedRepository) GetByID(id int64) (*models.Feed, error) {
	var f models.Feed
	var lastFetched, createdAt, lastRefreshed sql.NullString
	err := DB.QueryRow(
		`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at, COALESCE(group_name, ''),
	        last_refresh_success, COALESCE(last_refresh_error, ''), last_refreshed, unread_count
	 FROM feeds WHERE id = ?`,
		id,
	).Scan(&f.ID, &f.Title, &f.URL, &f.Description, &f.IconURL, &lastFetched, &f.IsDead, &createdAt, &f.Group,
		&f.LastRefreshSuccess, &f.LastRefreshError, &lastRefreshed, &f.UnreadCount)
	if err != nil {
		return nil, err
	}
	if lastFetched.Valid {
		f.LastFetched, _ = time.Parse(time.RFC3339, lastFetched.String)
	}
	if createdAt.Valid {
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	if lastRefreshed.Valid {
		f.LastRefreshed, _ = time.Parse(time.RFC3339, lastRefreshed.String)
	}
	return &f, nil
}

func (r *FeedRepository) Update(feed *models.Feed) error {
	_, err := DB.Exec(
		`UPDATE feeds SET title = ?, url = ?, description = ?, icon_url = ?, last_fetched = ?, group_name = ? WHERE id = ?`,
		feed.Title, feed.URL, feed.Description, feed.IconURL, feed.LastFetched.Format(time.RFC3339), feed.Group, feed.ID,
	)
	return err
}

func (r *FeedRepository) Delete(id int64) error {
	// Cascade delete: remove feed's articles first
	_, err := DB.Exec(`DELETE FROM articles WHERE feed_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM feeds WHERE id = ?`, id)
	return err
}

func (r *FeedRepository) UpdateRefreshResult(id int64, success int, errorMsg string) error {
	_, err := DB.Exec(
		`UPDATE feeds SET last_refresh_success = ?, last_refresh_error = ?, last_refreshed = ? WHERE id = ?`,
		success, errorMsg, time.Now().Format(time.RFC3339), id,
	)
	return err
}

func (r *FeedRepository) MarkDead(id int64) error {
	_, err := DB.Exec(`UPDATE feeds SET is_dead = 1 WHERE id = ?`, id)
	return err
}

func (r *FeedRepository) GetDeadFeeds() ([]models.Feed, error) {
	rows, err := DB.Query(`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at, COALESCE(group_name, '') FROM feeds WHERE is_dead = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds = []models.Feed{}
	for rows.Next() {
		var f models.Feed
		var lastFetched, createdAt sql.NullString
		err := rows.Scan(&f.ID, &f.Title, &f.URL, &f.Description, &f.IconURL, &lastFetched, &f.IsDead, &createdAt, &f.Group)
		if err != nil {
			log.Printf("scan dead feed error (row may be skipped): %v", err)
			continue
		}
		if lastFetched.Valid {
			f.LastFetched, _ = time.Parse(time.RFC3339, lastFetched.String)
		}
		if createdAt.Valid {
			f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}
		feeds = append(feeds, f)
	}
	return feeds, nil
}

// UpdateUnreadCount increments or decrements the unread count for a feed
func (r *FeedRepository) UpdateUnreadCount(feedId int64, delta int) error {
	_, err := DB.Exec(`UPDATE feeds SET unread_count = MAX(0, unread_count + ?) WHERE id = ?`, delta, feedId)
	return err
}

// RecalcUnreadCount recalculates unread_count from article table
func (r *FeedRepository) RecalcUnreadCount(feedId int64) error {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM articles WHERE feed_id = ? AND status = 'unread'`, feedId).Scan(&count)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`UPDATE feeds SET unread_count = ? WHERE id = ?`, count, feedId)
	return err
}
