package sqlite

import (
	"database/sql"
)

type SettingsRepository struct{}

func NewSettingsRepository() *SettingsRepository {
	return &SettingsRepository{}
}

func (r *SettingsRepository) Get(key string) (string, error) {
	var value string
	err := DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (r *SettingsRepository) Set(key, value string) error {
	_, err := DB.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?`,
		key, value, value,
	)
	return err
}

func (r *SettingsRepository) Delete(key string) error {
	_, err := DB.Exec(`DELETE FROM settings WHERE key = ?`, key)
	return err
}
