package models

import "time"

type GetDomainsDTO struct {
	TotalPages    int
	Page          int
	PageSize      int
	TotalElements int
	HasNext       bool
	HasPrev       bool
	NextPage      int
	PrevPage      int
	Domains       []DomainsDTO
}

type CertsDTO struct {
	ID              string
	Issuer          *string
	CertPath        string
	KeyPath         string
	ChainPath       *string
	ValidFrom       *time.Time
	ValidTo         *time.Time
	LastRenewal     *time.Time
	RenewalAttempts int
	CreatedAt       time.Time
	CreatedBy       string
}

type DomainsDTO struct {
	ID         string
	DomainName string
	Details    DetailsDTO
	Sub        []string
}

type DetailsDTO struct {
	DNSProvider         string
	Status              string
	AutoRenew           bool
	VerificationMethod  string
	CreatedAt           time.Time
	CreatedBy           string
	DomainLastUpdate    *time.Time
	NginxContainerName  string
	CertValidTo         *time.Time
	CertLastRenewal     *time.Time
	CertRenewalAttempts *int
}
