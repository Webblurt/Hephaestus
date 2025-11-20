package models

import "time"

type GetDomainsResp struct {
	TotalPages    int       `json:"total_pages"`
	Page          int       `json:"page"`
	PageSize      int       `json:"page_size"`
	TotalElements int       `json:"total_elements"`
	HasNext       bool      `json:"has_next"`
	HasPrev       bool      `json:"has_prev"`
	NextPage      int       `json:"next_page,omitempty"`
	PrevPage      int       `json:"prev_page,omitempty"`
	Domains       []Domains `json:"domains"`
}

type Domains struct {
	ID         string   `json:"id"`
	DomainName string   `json:"domain_name"`
	Details    Details  `json:"details"`
	Sub        []string `json:"sub"`
}

type Details struct {
	DNSProvider         string    `json:"dns_provider"`
	Status              string    `json:"status"`
	AutoRenew           bool      `json:"auto_renew"`
	VerificationMethod  string    `json:"verification_method"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           string    `json:"created_by"`
	DomainLastUpdate    time.Time `json:"domain_last_update"`
	NginxContainerName  string    `json:"nginx_container_name"`
	CertValidTo         time.Time `json:"certificate_valid_to"`
	CertLastRenewal     time.Time `json:"certificate_last_renewal"`
	CertRenewalAttempts int       `json:"certificate_renewal_attempts"`
}
