package repositories

import (
	"context"
)

func (r *Repository) GetListOfSubDomains(ctx context.Context, domainID string) ([]string, error) {
	r.log.Debug("Filters in repo layer: ", domainID)

	query := `
		SELECT id
		FROM alternative_domains
		WHERE deleted_at IS NULL
		AND domain_id = $1
	`
	rows, err := r.DB.Query(ctx, query, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subDomains []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		subDomains = append(subDomains, id)
	}

	return subDomains, nil
}
