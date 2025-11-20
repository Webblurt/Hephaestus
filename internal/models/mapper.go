package models

import "time"

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeStrings(ptrs []*string) []string {
	if len(ptrs) == 0 {
		return nil
	}
	res := make([]string, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			res = append(res, *p)
		}
	}
	return res
}

func safeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func safeTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func ConvertDomainsDTOToDomains(req DomainsDTO) Domains {
	return Domains{
		ID:         req.ID,
		DomainName: req.DomainName,
		Sub:        req.Sub,
		Details: Details{
			DNSProvider:         req.Details.DNSProvider,
			Status:              req.Details.Status,
			AutoRenew:           req.Details.AutoRenew,
			VerificationMethod:  req.Details.VerificationMethod,
			CreatedAt:           req.Details.CreatedAt,
			CreatedBy:           req.Details.CreatedBy,
			DomainLastUpdate:    safeTime(req.Details.DomainLastUpdate),
			NginxContainerName:  req.Details.NginxContainerName,
			CertValidTo:         safeTime(req.Details.CertValidTo),
			CertLastRenewal:     safeTime(req.Details.CertLastRenewal),
			CertRenewalAttempts: safeInt(req.Details.CertRenewalAttempts),
		},
	}
}
