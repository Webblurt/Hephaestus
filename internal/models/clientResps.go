package models

import "time"

type CertificateData struct {
	Cert      []byte
	Key       []byte
	Chain     []byte
	ValidFrom time.Time
	ValidTo   time.Time
}

type CertificatePaths struct {
	Cert  string
	Key   string
	Chain string
}
