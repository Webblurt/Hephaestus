package repositories

import (
	"context"
	"fmt"
	models "hephaestus/internal/models"
)

func (r *Repository) GetCertificatesByDomain(ctx context.Context, domainID string) (models.CertsDTO, error) {
	r.log.Debug("id in repo layer: ", domainID)
	query := `
        SELECT 
            id, issuer, cert_path, key_path, chain_path, valid_from,
			valid_to, last_renewal, renewal_attempts, created_at, created_by
        FROM certificates 
        WHERE deleted_at IS NULL
    `

	args := []interface{}{}
	argID := 1

	query += fmt.Sprintf(" AND domain_id = $%d", argID)
	args = append(args, "%"+domainID+"%")
	argID++

	r.log.Debug("Query execution: ", query)
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return models.CertsDTO{}, err
	}
	r.log.Debug("Query executed.")
	defer rows.Close()

	var certs models.CertsDTO
	r.log.Debug("Starting to scan rows")

	err = rows.Scan(
		&certs.ID, &certs.Issuer, &certs.CertPath, &certs.KeyPath, &certs.ChainPath, &certs.ValidFrom,
		&certs.ValidTo, &certs.LastRenewal, &certs.RenewalAttempts, &certs.CreatedAt, &certs.CreatedBy,
	)
	if err != nil {
		return certs, err

	}
	r.log.Debug("Rows scanned.")

	if err = rows.Err(); err != nil {
		return certs, err
	}

	return certs, nil
}
