package repositories

import (
	"context"
	"fmt"
	models "hephaestus/internal/models"
)

func (r *Repository) IsDomainExists(ctx context.Context, domain string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM domains WHERE domain_name = $1);`

	var exists bool
	err := r.DB.QueryRow(ctx, query, domain).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *Repository) GetDomainsCount(ctx context.Context, filters models.DomainsFilters) (int, error) {
	r.log.Debug("Filters in repo layer: ", filters)

	query := `
		SELECT COUNT(*)
		FROM domains
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	argID := 1

	if filters.DomainName != "" {
		query += fmt.Sprintf(" AND domain_name ILIKE $%d", argID)
		args = append(args, "%"+filters.DomainName+"%")
		argID++
	}
	if filters.Status != "" {
		query += fmt.Sprintf(" AND status ILIKE $%d", argID)
		args = append(args, "%"+filters.Status+"%")
		argID++
	}
	if filters.UserID != "" {
		query += fmt.Sprintf(" AND created_by = $%d", argID)
		args = append(args, filters.UserID)
		argID++
	}

	var count int
	r.log.Debug("Query execution: ", query)
	err := r.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	r.log.Debug("Query executed.")
	r.log.Debug("Total itineraries: ", count)

	return count, nil
}

func (r *Repository) GetDomainsList(ctx context.Context, filters models.DomainsFilters) ([]models.DomainsDTO, error) {
	r.log.Debug("Filters in repo layer: ", filters)

	subQuery := `
		SELECT d.id
		FROM domains d
		WHERE d.deleted_at IS NULL
	`
	args := []interface{}{}
	argID := 1

	if filters.DomainName != "" {
		subQuery += fmt.Sprintf(" AND d.domain_name ILIKE $%d", argID)
		args = append(args, "%"+filters.DomainName+"%")
		argID++
	}
	if filters.Status != "" {
		subQuery += fmt.Sprintf(" AND d.status ILIKE $%d", argID)
		args = append(args, "%"+filters.Status+"%")
		argID++
	}
	if filters.UserID != "" {
		subQuery += fmt.Sprintf(" AND d.created_by = $%d", argID)
		args = append(args, filters.UserID)
		argID++
	}

	if filters.Limit != nil && filters.Offset != nil {
		subQuery += fmt.Sprintf(" ORDER BY d.created_at DESC LIMIT $%d OFFSET $%d", argID, argID+1)
		args = append(args, filters.Limit, filters.Offset)
		argID += 2
	}

	query := fmt.Sprintf(`
		SELECT 
			d.id, d.domain_name, d.dns_provider, d.status, d.auto_renew,
			d.nginx_container_name, d.verification_method, d.created_at, d.created_by, d.updated_at, 
			c.valid_to, c.last_renewal, c.renewal_attempts,
			COALESCE(
				array_agg(ad.domain_name) FILTER (WHERE ad.domain_name IS NOT NULL),
				'{}'
			) AS subdomains
		FROM (%s) AS domains_list
		JOIN domains d ON d.id = domains_list.id
		LEFT JOIN certificates c ON c.domain_id = d.id AND c.deleted_at IS NULL
		LEFT JOIN alternative_domains ad ON ad.domain_id = d.id AND ad.deleted_at IS NULL
		GROUP BY 
			d.id, d.domain_name, d.dns_provider, d.status, d.auto_renew,
			d.nginx_container_name, d.verification_method, d.created_at, d.created_by, d.updated_at,
			c.valid_to, c.last_renewal, c.renewal_attempts
		ORDER BY d.id;
		`, subQuery)

	r.log.Debug("Query execution: ", query)
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r.log.Debug("Query executed.")

	var domains []models.DomainsDTO

	for rows.Next() {
		var domain models.DomainsDTO

		err := rows.Scan(
			&domain.ID, &domain.DomainName, &domain.Details.DNSProvider, &domain.Details.Status,
			&domain.Details.AutoRenew, &domain.Details.NginxContainerName, &domain.Details.VerificationMethod,
			&domain.Details.CreatedAt, &domain.Details.CreatedBy, &domain.Details.DomainLastUpdate,
			&domain.Details.CertValidTo, &domain.Details.CertLastRenewal, &domain.Details.CertRenewalAttempts,
			&domain.Sub,
		)
		if err != nil {
			return nil, err
		}

		domains = append(domains, domain)
	}

	return domains, nil
}
