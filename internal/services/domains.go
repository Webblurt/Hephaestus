package services

import (
	"fmt"
	models "hephaestus/internal/models"
	"time"
)

func (s *Service) GetDomains(filters models.GetDomainsReq) (models.GetDomainsResp, error) {
	s.log.Debug("Fetching list of domains started............")
	offset := (filters.Page - 1) * filters.PageSize
	repoFilters := models.DomainsFilters{
		DomainName: filters.DomainName,
		Status:     filters.Status,
		UserID:     filters.UserID,
		Limit:      &filters.PageSize,
		Offset:     &offset,
	}

	s.log.Debug("Fetching stocks count from repo...")
	totalElements, err := s.repository.GetDomainsCount(s.ctx, repoFilters)
	if err != nil {
		s.log.Error("Error while getting total stock count: ", err)
		return models.GetDomainsResp{}, err
	}
	s.log.Debug("Count: ", totalElements)

	totalPages := (totalElements + filters.Page - 1) / filters.PageSize
	hasNext := filters.Page < totalPages
	hasPrev := filters.Page > 1
	nextPage := 0
	prevPage := 0
	if hasNext {
		nextPage = filters.Page + 1
	}
	if hasPrev {
		prevPage = filters.Page - 1
	}

	s.log.Debug("Fetching list of domains from repo...")
	domains, err := s.repository.GetDomainsList(s.ctx, repoFilters)
	if err != nil {
		s.log.Error("Error while getting list of domains: ", err)
		return models.GetDomainsResp{}, err
	}
	s.log.Debug("List of domains: ", domains)

	var d []models.Domains
	for _, domain := range domains {
		d = append(d, models.ConvertDomainsDTOToDomains(domain))
	}

	return models.GetDomainsResp{
		TotalPages:    totalPages,
		Page:          filters.Page,
		PageSize:      filters.PageSize,
		TotalElements: totalElements,
		HasNext:       hasNext,
		HasPrev:       hasPrev,
		NextPage:      nextPage,
		PrevPage:      prevPage,
		Domains:       d,
	}, nil
}

func (s *Service) CreateDomain(req models.CreateDomainReq) (domainID string, err error) {
	s.log.Debug("CreateDomain: start")

	exists, err := s.repository.IsDomainExists(s.ctx, req.Domain)
	if err != nil {
		return "", fmt.Errorf("check domain exists: %w", err)
	}
	if exists {
		return "", fmt.Errorf("domain already exists")
	}

	client, err := s.SelectClientByName(req.DNSProvider)
	if err != nil {
		return "", fmt.Errorf("select DNS client: %w", err)
	}

	certData, err := client.CreateCertificate(req.Domain, req.AltDomains)
	if err != nil {
		s.log.Error("certificate creation failed:", err)
		_ = s.safeWriteEvent(req.CreatedBy, "", "failed",
			fmt.Sprintf("Certificate creation failed: %v", err))
		return "", fmt.Errorf("certificate creation failed: %w", err)
	}

	certPaths, err := client.SaveCertificateFiles(req.Domain, certData)
	if err != nil {
		s.log.Error("saving certificate files failed:", err)
		_ = s.safeWriteEvent(req.CreatedBy, "", "failed",
			fmt.Sprintf("Saving certificate files failed: %v", err))
		return "", fmt.Errorf("save certificate files: %w", err)
	}

	tx, err := s.repository.BeginTx(s.ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			s.log.Warn("CreateDomain: rollback due to error:", err)
			_ = tx.Rollback(s.ctx)

			_ = s.safeWriteEvent(
				req.CreatedBy,
				domainID,
				"failed",
				fmt.Sprintf("Domain creation failed: %v", err),
			)
		}
	}()

	domainEntity := NewEntity("domains", map[string]any{
		"domain_name":          req.Domain,
		"dns_provider":         req.DNSProvider,
		"status":               "pending",
		"verification_method":  req.VerificationMethod,
		"nginx_container_name": req.NginxContainerName,
		"created_by":           req.CreatedBy,
		"auto_renew":           req.AutoRenew,
	})

	domainID, err = s.repository.InsertTx(s.ctx, tx, domainEntity)
	if err != nil {
		return "", fmt.Errorf("insert domain: %w", err)
	}

	for _, sub := range req.AltDomains {
		subEntity := NewEntity("alternative_domains", map[string]any{
			"domain_id":   domainID,
			"domain_name": sub,
			"created_by":  req.CreatedBy,
		})

		_, err = s.repository.InsertTx(s.ctx, tx, subEntity)
		if err != nil {
			return "", fmt.Errorf("insert alt domain '%s': %w", sub, err)
		}
	}

	certEntity := NewEntity("certificates", map[string]any{
		"domain_id":  domainID,
		"issuer":     "Let's Encrypt",
		"cert_path":  certPaths.Cert,
		"key_path":   certPaths.Key,
		"chain_path": certPaths.Chain,
		"created_by": req.CreatedBy,
		"valid_from": certData.ValidFrom,
		"valid_to":   certData.ValidTo,
	})

	_, err = s.repository.InsertTx(s.ctx, tx, certEntity)
	if err != nil {
		return "", fmt.Errorf("insert certificate: %w", err)
	}

	err = s.updateMany(s.ctx, tx, map[string]models.Entity{
		domainID: NewEntity("domains", map[string]any{
			"status":     "active",
			"updated_by": req.CreatedBy,
		}),
	})
	if err != nil {
		return "", fmt.Errorf("update domain status: %w", err)
	}

	// Commit
	if err = tx.Commit(s.ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}

	_ = s.safeWriteEvent(
		req.CreatedBy,
		domainID,
		"created",
		"Domain and certificate created successfully",
	)

	s.log.Debug("CreateDomain: success")
	return domainID, nil
}

func (s *Service) DeleteDomain(filters models.DeleteDomainReq) (err error) {
	s.log.Debug("Deleting domain...")

	// start transaction
	tx, err := s.repository.BeginTx(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// rollback on error
	defer func() {
		if err != nil {
			s.log.Warn("Rollback started")
			if rollbackErr := tx.Rollback(s.ctx); rollbackErr != nil {
				s.log.Error("Rollback error:", rollbackErr)
			}
		}
	}()

	// determine domain ID
	domainID := filters.DomainID
	if filters.DomainName != "" {
		domainID, err = s.repository.GetIDByNameTx(s.ctx, tx, models.Entity{
			EntityName:       "domains",
			StringParameters: map[string]string{"domain_name": filters.DomainName},
		})
		if err != nil {
			return fmt.Errorf("error while getting domain id: %w", err)
		}
		if domainID == "" {
			return fmt.Errorf("domain doesn't exist")
		}
	}

	// fetch subdomains
	subDomains, err := s.repository.GetListOfSubDomains(s.ctx, domainID)
	if err != nil {
		return fmt.Errorf("error getting subdomains: %w", err)
	}

	now := time.Now()

	// prepare update data for domains
	updateDomains := make(map[string]models.Entity, 1+len(subDomains))

	// main domain
	updateDomains[domainID] = NewEntity("domains", map[string]any{
		"status":     "deleted",
		"deleted_by": filters.UserID,
		"updated_by": filters.UserID,
		"deleted_at": now,
	})

	// subdomains
	for _, id := range subDomains {
		updateDomains[id] = NewEntity("alternative_domains", map[string]any{
			"status":     "deleted",
			"deleted_by": filters.UserID,
			"updated_by": filters.UserID,
			"deleted_at": now,
		})
	}

	if err = s.updateMany(s.ctx, tx, updateDomains); err != nil {
		return fmt.Errorf("error updating domain statuses: %w", err)
	}

	// fetch certificates
	certs, err := s.repository.GetCertificatesByDomain(s.ctx, domainID)
	if err != nil {
		return fmt.Errorf("error fetching certificates: %w", err)
	}

	// mark certificates as deleted
	if certs.ID != "" {
		err = s.updateMany(s.ctx, tx, map[string]models.Entity{
			certs.ID: NewEntity("certificates", map[string]any{
				"deleted_by": filters.UserID,
				"updated_by": filters.UserID,
				"deleted_at": now,
			}),
		})
		if err != nil {
			return fmt.Errorf("error updating certificate: %w", err)
		}
	}

	// write event
	if err = s.writeEvent(
		s.ctx, tx, domainID, "deleted",
		fmt.Sprintf("Domain '%s' and its certificates deleted", filters.DomainName),
		filters.UserID,
	); err != nil {
		return fmt.Errorf("error inserting event: %w", err)
	}

	// COMMIT TRANSACTION
	if err = tx.Commit(s.ctx); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	// delete files safely AFTER commit
	go func(domain string) {
		client, cErr := s.SelectClientByName("no")
		if cErr != nil {
			s.log.Warn("Error selecting client:", cErr)
			return
		}
		if dErr := client.DeleteCertificateFiles(domain); dErr != nil {
			s.log.Warn("Error deleting certificate files:", dErr)
		}
	}(filters.DomainName)

	s.log.Debug("Domain deleted successfully")
	return nil
}
