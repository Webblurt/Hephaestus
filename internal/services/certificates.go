package services

import (
	"fmt"
	models "hephaestus/internal/models"
	"os/exec"
	"time"
)

func (s *Service) StartCertificateRenewalScheduler() {
	ticker := time.NewTicker(s.cfg.Certs.RenewalDuration * time.Hour)

	go func() {
		for range ticker.C {
			s.log.Info("Running certificate renewal cycle...")
			s.RenewExpiringCertificates()
		}
	}()
}

func (s *Service) RenewExpiringCertificates() {
	domains, err := s.repository.GetDomainsList(s.ctx, models.DomainsFilters{})
	if err != nil {
		s.log.Error("failed fetch domains:", err)
		return
	}

	now := time.Now()

	for _, d := range domains {

		if d.Details.Status == "deleted" || !d.Details.AutoRenew {
			continue
		}

		// renew threshold: 30 days before expiration
		renewDate := d.Details.CertValidTo.Add(-30 * 24 * time.Hour)

		if now.Before(renewDate) {
			continue
		}

		s.log.Info("Domain %s is approaching expiration (%s). Renewal triggered.",
			d.DomainName, d.Details.CertValidTo.Format(time.RFC3339))

		if err := s.RenewDomainCertificate(d); err != nil {
			s.log.Error("Failed to renew certificate for", d.DomainName, ":", err)
		}
	}
}

func (s *Service) RenewDomainCertificate(domain models.DomainsDTO) error {
	s.log.Info("Renewing certificate for domain: ", domain.DomainName)

	tx, err := s.repository.BeginTx(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			s.log.Warn("Rollback renewal tx")
			_ = tx.Rollback(s.ctx)
		}
	}()

	s.log.Debug("Selecting client...")
	client, err := s.SelectClientByName(domain.Details.DNSProvider)

	var san []string
	if len(domain.Sub) > 0 {
		san = domain.Sub
	}

	// request
	certData, err := client.CreateCertificate(domain.DomainName, san)
	if err != nil {
		s.log.Error("renewal certificate failed:", err)
		updateDomainsData := make(map[string]models.Entity)
		updateDomainsData[domain.ID] = NewEntity("domains", map[string]any{
			"status":     "update_failed",
			"updated_by": "system-renewal",
		})
		err := s.updateMany(s.ctx, tx, updateDomainsData)
		err = s.writeEvent(s.ctx, tx, domain.ID, "failed", fmt.Sprintf("Certificate issuance failed: %v", err), "system-renewal")
		return fmt.Errorf("failed to create new certificate: %w", err)
	}

	// saving files
	certPaths, err := client.SaveCertificateFiles(domain.DomainName, certData)
	if err != nil {
		updateDomainsData := make(map[string]models.Entity)
		updateDomainsData[domain.ID] = NewEntity("domains", map[string]any{
			"status":     "update_failed",
			"updated_by": "system-renewal",
		})
		_ = s.updateMany(s.ctx, tx, updateDomainsData)
		return fmt.Errorf("failed to save cert files: %w", err)
	}

	// updatind db certs
	updateCertData := map[string]models.Entity{
		domain.ID: NewEntity("certificates", map[string]any{
			"cert_path":  certPaths.Cert,
			"key_path":   certPaths.Key,
			"chain_path": certPaths.Chain,
			"updated_by": "system-renewal",
			"valid_from": certData.ValidFrom,
			"valid_to":   certData.ValidTo,
			"updated_at": time.Now(),
		}),
	}
	err = s.updateMany(s.ctx, tx, updateCertData)
	if err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	updateDomainsData := make(map[string]models.Entity)
	updateDomainsData[domain.ID] = NewEntity("domains", map[string]any{
		"status":     "active",
		"updated_by": "system-renewal",
	})
	err = s.updateMany(s.ctx, tx, updateDomainsData)
	if err != nil {
		return fmt.Errorf("failed to update domain status: %w", err)
	}

	// event
	err = s.writeEvent(s.ctx, tx, domain.ID, "renewed", fmt.Sprintf("Certificate for '%s' renewed", domain.DomainName), "system-renewal")
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	err = tx.Commit(s.ctx)
	if err != nil {
		return fmt.Errorf("failed commit: %w", err)
	}

	s.log.Info("Domain %s successfully renewed!", domain.DomainName)

	if err := s.reloadNginxInContainer(domain); err != nil {
		return fmt.Errorf("certificate renewed but nginx reload failed: %w", err)
	}

	return nil
}

func (s *Service) reloadNginxInContainer(domain models.DomainsDTO) error {
	if domain.Details.NginxContainerName == "" {
		return nil
	}

	cmd := exec.Command("docker", "exec", domain.Details.NginxContainerName, "nginx", "-s", "reload")
	out, err := cmd.CombinedOutput()

	if err != nil {
		s.log.Error("Docker nginx reload error:", string(out))
		return fmt.Errorf("failed to reload nginx inside container: %w", err)
	}

	s.log.Info("Nginx inside container reloaded for domain:", domain.DomainName)
	return nil
}
