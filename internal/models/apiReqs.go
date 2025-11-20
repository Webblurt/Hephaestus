package models

type GetDomainsReq struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	UserID     string
	Status     string `json:"status,omitempty"`
	DomainName string `json:"domain_name,omitempty"`
}

type CreateDomainReq struct {
	CreatedBy          string
	Domain             string   `json:"domain"`
	AltDomains         []string `json:"alternative_domains"`
	VerificationMethod string   `json:"verification_method"`
	AutoRenew          bool     `json:"auto_renew"`
	NginxContainerName string   `json:"nginx_container_name"`
	DNSProvider        string   `json:"dns_provider"`
}

type DeleteDomainReq struct {
	DomainID   string `json:"domain_id"`
	DomainName string `json:"domain_name"`
	UserID     string
}
