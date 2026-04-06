package sqlite

import (
	"ai-rss-reader/internal/models"
	"database/sql"
)

type PromptRepository struct{}

func NewPromptRepository() *PromptRepository {
	return &PromptRepository{}
}

func (r *PromptRepository) GetAll() ([]models.PromptConfig, error) {
	rows, err := DB.Query(`SELECT id, type, name, prompt, system, max_tokens, is_default FROM prompt_configs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []models.PromptConfig
	for rows.Next() {
		var p models.PromptConfig
		err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.Prompt, &p.System, &p.MaxTokens, &p.IsDefault)
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}

func (r *PromptRepository) GetByID(id int64) (*models.PromptConfig, error) {
	var p models.PromptConfig
	err := DB.QueryRow(
		`SELECT id, type, name, prompt, system, max_tokens, is_default FROM prompt_configs WHERE id = ?`,
		id,
	).Scan(&p.ID, &p.Type, &p.Name, &p.Prompt, &p.System, &p.MaxTokens, &p.IsDefault)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PromptRepository) GetByType(promptType string) (*models.PromptConfig, error) {
	var p models.PromptConfig
	err := DB.QueryRow(
		`SELECT id, type, name, prompt, system, max_tokens, is_default FROM prompt_configs WHERE type = ?`,
		promptType,
	).Scan(&p.ID, &p.Type, &p.Name, &p.Prompt, &p.System, &p.MaxTokens, &p.IsDefault)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PromptRepository) GetDefault(promptType string) (*models.PromptConfig, error) {
	var p models.PromptConfig
	err := DB.QueryRow(
		`SELECT id, type, name, prompt, system, max_tokens, is_default FROM prompt_configs WHERE type = ? AND is_default = 1`,
		promptType,
	).Scan(&p.ID, &p.Type, &p.Name, &p.Prompt, &p.System, &p.MaxTokens, &p.IsDefault)
	if err == sql.ErrNoRows {
		// Fallback to first one of this type
		err = DB.QueryRow(
			`SELECT id, type, name, prompt, system, max_tokens, is_default FROM prompt_configs WHERE type = ? LIMIT 1`,
			promptType,
		).Scan(&p.ID, &p.Type, &p.Name, &p.Prompt, &p.System, &p.MaxTokens, &p.IsDefault)
		if err == sql.ErrNoRows {
			return nil, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PromptRepository) Create(p *models.PromptConfig) error {
	result, err := DB.Exec(
		`INSERT INTO prompt_configs (type, name, prompt, system, max_tokens, is_default) VALUES (?, ?, ?, ?, ?, ?)`,
		p.Type, p.Name, p.Prompt, p.System, p.MaxTokens, boolToInt(p.IsDefault),
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	p.ID = id
	return nil
}

func (r *PromptRepository) Update(p *models.PromptConfig) error {
	_, err := DB.Exec(
		`UPDATE prompt_configs SET name = ?, prompt = ?, system = ?, max_tokens = ?, is_default = ? WHERE id = ?`,
		p.Name, p.Prompt, p.System, p.MaxTokens, boolToInt(p.IsDefault), p.ID,
	)
	return err
}

func (r *PromptRepository) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM prompt_configs WHERE id = ?`, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
