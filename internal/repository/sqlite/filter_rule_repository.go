package sqlite

import (
	"ai-rss-reader/internal/models"
)

type FilterRuleRepository struct{}

func NewFilterRuleRepository() *FilterRuleRepository {
	return &FilterRuleRepository{}
}

func (r *FilterRuleRepository) Create(rule *models.FilterRule) error {
	result, err := DB.Exec(
		`INSERT INTO filter_rules (type, value, action, enabled, created_at) VALUES (?, ?, ?, ?, ?)`,
		rule.Type, rule.Value, rule.Action, rule.Enabled, rule.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	rule.ID = id
	return nil
}

func (r *FilterRuleRepository) GetAll() ([]models.FilterRule, error) {
	rows, err := DB.Query(`SELECT id, type, value, action, enabled, created_at FROM filter_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules = []models.FilterRule{}
	for rows.Next() {
		var rule models.FilterRule
		err := rows.Scan(&rule.ID, &rule.Type, &rule.Value, &rule.Action, &rule.Enabled, &rule.CreatedAt)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *FilterRuleRepository) GetEnabled() ([]models.FilterRule, error) {
	rows, err := DB.Query(`SELECT id, type, value, action, enabled, created_at FROM filter_rules WHERE enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules = []models.FilterRule{}
	for rows.Next() {
		var rule models.FilterRule
		err := rows.Scan(&rule.ID, &rule.Type, &rule.Value, &rule.Action, &rule.Enabled, &rule.CreatedAt)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *FilterRuleRepository) Update(rule *models.FilterRule) error {
	_, err := DB.Exec(
		`UPDATE filter_rules SET type = ?, value = ?, action = ?, enabled = ? WHERE id = ?`,
		rule.Type, rule.Value, rule.Action, rule.Enabled, rule.ID,
	)
	return err
}

func (r *FilterRuleRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM filter_rules WHERE id = ?`, id)
	return err
}
