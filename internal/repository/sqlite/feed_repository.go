package sqlite

import (
	"ai-rss-reader/internal/models"
	"database/sql"
	"time"
)

type FeedRepository struct{}

func NewFeedRepository() *FeedRepository {
	return &FeedRepository{}
}

func (r *FeedRepository) Create(feed *models.Feed) error {
	result, err := DB.Exec(
		`INSERT INTO feeds (title, url, description, icon_url, last_fetched, is_dead, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		feed.Title, feed.URL, feed.Description, feed.IconURL, feed.LastFetched.Format(time.RFC3339), false, feed.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	feed.ID = id
	return nil
}

func (r *FeedRepository) GetAll() ([]models.Feed, error) {
	rows, err := DB.Query(`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at FROM feeds ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var f models.Feed
		var lastFetched, createdAt sql.NullString
		err := rows.Scan(&f.ID, &f.Title, &f.URL, &f.Description, &f.IconURL, &lastFetched, &createdAt)
		if err != nil {
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

func (r *FeedRepository) GetByID(id int64) (*models.Feed, error) {
	var f models.Feed
	var lastFetched, createdAt sql.NullString
	err := DB.QueryRow(
		`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at FROM feeds WHERE id = ?`,
		id,
	).Scan(&f.ID, &f.Title, &f.URL, &f.Description, &f.IconURL, &lastFetched, &f.IsDead, &createdAt)
	if err != nil {
		return nil, err
	}
	if lastFetched.Valid {
		f.LastFetched, _ = time.Parse(time.RFC3339, lastFetched.String)
	}
	if createdAt.Valid {
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}
	return &f, nil
}

func (r *FeedRepository) Update(feed *models.Feed) error {
	_, err := DB.Exec(
		`UPDATE feeds SET title = ?, description = ?, icon_url = ?, last_fetched = ? WHERE id = ?`,
		feed.Title, feed.Description, feed.IconURL, feed.LastFetched.Format(time.RFC3339), feed.ID,
	)
	return err
}

func (r *FeedRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM feeds WHERE id = ?`, id)
	return err
}

func (r *FeedRepository) MarkDead(id int64) error {
	_, err := DB.Exec(`UPDATE feeds SET is_dead = 1 WHERE id = ?`, id)
	return err
}

func (r *FeedRepository) GetDeadFeeds() ([]models.Feed, error) {
	rows, err := DB.Query(`SELECT id, title, url, description, icon_url, last_fetched, is_dead, created_at FROM feeds WHERE is_dead = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var f models.Feed
		var lastFetched, createdAt sql.NullString
		err := rows.Scan(&f.ID, &f.Title, &f.URL, &f.Description, &f.IconURL, &lastFetched, &f.IsDead, &createdAt)
		if err != nil {
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
