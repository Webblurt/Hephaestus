package clients

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	cf "github.com/go-acme/lego/v4/providers/dns/cloudflare"
	dod "github.com/go-acme/lego/v4/providers/dns/digitalocean"
	hz "github.com/go-acme/lego/v4/providers/dns/hetzner"
	r53 "github.com/go-acme/lego/v4/providers/dns/route53"

	models "hephaestus/internal/models"
	utils "hephaestus/internal/utils"
)

type DNSProvider interface {
	CreateTXTRecord(ctx context.Context, domain, name, value string, ttl int) error
	DeleteTXTRecord(ctx context.Context, domain, name, value string) error
}

type Client struct {
	Name         string
	URL          string
	Key          string
	DNS          DNSProvider
	legoProvider challenge.Provider // underlying lego DNS provider for SetDNS01Provider
	Manager      *autocertShim
	log          *utils.Logger
	cfg          *utils.Config
	acmeUserKey  crypto.PrivateKey
}

type autocertShim struct{}

func NewClient(name, url, key string, log *utils.Logger, cfg *utils.Config) (*Client, error) {
	log.Debug("NewClient(): called",
		" name=", name,
		" url=", url,
		" keyExists=", key != "",
	)
	c := &Client{
		Name:    name,
		URL:     url,
		Key:     key,
		log:     log,
		cfg:     cfg,
		Manager: &autocertShim{},
	}

	log.Debug("Ensuring storage directory exists: ", cfg.Certs.StorageDir)
	// ensure storage dir exists
	if err := os.MkdirAll(cfg.Certs.StorageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to ensure storage dir: %w", err)
	}

	// load or create ACME user key
	keyPath := filepath.Join(cfg.Certs.StorageDir, "acme_user.key")
	log.Debug("Loading or creating ACME user key: ", keyPath)
	priv, err := c.loadOrCreatePrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load/create acme user key: %w", err)
	}
	log.Debug("ACME user key successfully loaded")
	c.acmeUserKey = priv

	// create lego DNS provider and a DNSProvider wrapper that matches interface
	log.Debug("Initializing DNS provider: ", name)
	switch strings.ToLower(name) {
	case "cloudflare":
		log.Debug("Setting CLOUDFLARE_API_TOKEN env")
		os.Setenv("CLOUDFLARE_API_TOKEN", key)
		p, err := cf.NewDNSProvider()
		if err != nil {
			return nil, fmt.Errorf("cloudflare provider init: %w", err)
		}
		log.Debug("Cloudflare DNS provider init successful")
		c.legoProvider = p
		c.DNS = &legoDNSWrapper{prov: p}

	case "hetzner":
		log.Debug("Setting HETZNER_API_TOKEN env")
		os.Setenv("HETZNER_API_KEY", key)
		p, err := hz.NewDNSProvider()
		if err != nil {
			return nil, fmt.Errorf("hetzner provider init: %w", err)
		}
		log.Debug("Hetzner DNS provider init successful")
		c.legoProvider = p
		c.DNS = &legoDNSWrapper{prov: p}

	case "digitalocean":
		log.Debug("Setting DO_AUTH_TOKEN env")
		os.Setenv("DO_AUTH_TOKEN", key)
		p, err := dod.NewDNSProvider()
		if err != nil {
			return nil, fmt.Errorf("digitalocean provider init: %w", err)
		}
		log.Debug("DigitalOcean DNS provider init successful")
		c.legoProvider = p
		c.DNS = &legoDNSWrapper{prov: p}

	case "route53":
		log.Debug("Setting AWS credentials env")
		os.Setenv("AWS_ACCESS_KEY_ID", cfg.AwsConfig.AccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.AwsConfig.SecretKey)
		os.Setenv("AWS_REGION", cfg.AwsConfig.Region)

		p, err := r53.NewDNSProvider()
		if err != nil {
			return nil, fmt.Errorf("route53 provider init: %w", err)
		}
		log.Debug("Route53 DNS provider init successful")
		c.legoProvider = p
		c.DNS = &legoDNSWrapper{prov: p}

	default:
		log.Error("Unknown DNS provider: ", name)
		return nil, fmt.Errorf("unknown DNS provider: %s", name)
	}

	log.Debug("NewClient(): success for provider ", name)
	return c, nil
}

func CreateClients(cfg *utils.Config, log *utils.Logger) ([]*Client, error) {
	var clients []*Client
	for _, api := range cfg.APIS {
		if api.Name == "" || api.URL == "" {
			log.Warn("api configuration missing something")
			continue
		}
		client, err := NewClient(api.Name, api.URL, api.Key, log, cfg)
		if err != nil {
			log.Error("failed to create client: ", err)
			continue
		}
		clients = append(clients, client)
	}
	if len(clients) == 0 {
		return nil, errors.New("0 clients created")
	}
	return clients, nil
}

type LegoUser struct {
	Email        string
	Registration *registration.Resource
	PrivateKey   crypto.PrivateKey
}

func (u *LegoUser) GetEmail() string                        { return u.Email }
func (u *LegoUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *LegoUser) GetPrivateKey() crypto.PrivateKey        { return u.PrivateKey }

func (c *Client) CreateCertificate(domain string, san []string) (*models.CertificateData, error) {
	c.log.Debug("CreateCertificate(): called",
		" domain=", domain,
		" SAN=", san,
	)

	// prepare user
	c.log.Debug("Preparing LegoUser with email: ", c.cfg.Certs.Email)
	user := &LegoUser{
		Email:      c.cfg.Certs.Email,
		PrivateKey: c.acmeUserKey,
	}

	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryProduction
	config.Certificate.KeyType = certcrypto.RSA2048

	c.log.Debug("Creating lego client with CADir: ", config.CADirURL)
	lg, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %w", err)
	}

	// REGISTER ACME ACCOUNT (required)
	c.log.Debug("Registering ACME account...")

	reg, err := lg.Registration.Register(registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register acme account: %w", err)
	}
	user.Registration = reg

	// set DNS provider
	c.log.Debug("Setting DNS provider...")
	if c.legoProvider == nil {
		return nil, fmt.Errorf("lego DNS provider not configured for client %s", c.Name)
	}
	if err := lg.Challenge.SetDNS01Provider(c.legoProvider); err != nil {
		return nil, fmt.Errorf("failed to set dns provider: %w", err)
	}

	// domains list (unique)
	domains := uniqueDomains(append([]string{domain}, san...))
	c.log.Debug("Final domain list for certificate: ", domains)

	req := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	c.log.Debug("Requesting certificate from ACME...")
	certRes, err := lg.Certificate.Obtain(req)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain certificate: %w", err)
	}
	c.log.Info("Certificate obtained. Parsing validity...")

	// parse cert to get validity
	blocks, err := certcrypto.ParsePEMBundle(certRes.Certificate)
	var validFrom, validTo time.Time
	if err == nil && len(blocks) > 0 {
		validFrom = blocks[0].NotBefore
		validTo = blocks[0].NotAfter
		c.log.Debug("Parsed certificate validity: ",
			" from=", validFrom,
			" to=", validTo,
		)
	} else {
		// fallback
		c.log.Warn("Failed to parse certificate validity: ", err)
		validFrom = time.Now().UTC()
		validTo = validFrom.Add(90 * 24 * time.Hour)
	}

	data := &models.CertificateData{
		Cert:      certRes.Certificate,
		Key:       certRes.PrivateKey,
		Chain:     certRes.IssuerCertificate,
		ValidFrom: validFrom,
		ValidTo:   validTo,
	}

	c.log.Debug("CreateCertificate(): completed successfully")
	return data, nil
}

func (c *Client) SaveCertificateFiles(domain string, certData *models.CertificateData) (*models.CertificatePaths, error) {
	c.log.Debug("SaveCertificateFiles(): called for domain: ", domain)
	baseDir := filepath.Join(c.cfg.Certs.StorageDir, domain)
	c.log.Debug("Ensuring domain directory: ", baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create domain dir: %w", err)
	}

	certPath := filepath.Join(baseDir, "cert.pem")
	keyPath := filepath.Join(baseDir, "privkey.pem")
	chainPath := filepath.Join(baseDir, "chain.pem")

	c.log.Debug("Writing cert file: ", certPath)
	if err := os.WriteFile(certPath, certData.Cert, 0644); err != nil {
		return nil, fmt.Errorf("write cert: %w", err)
	}
	c.log.Debug("Writing key file: ", keyPath)
	if err := os.WriteFile(keyPath, certData.Key, 0600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}
	c.log.Debug("Writing chain file: ", chainPath)
	if len(certData.Chain) > 0 {
		if err := os.WriteFile(chainPath, certData.Chain, 0644); err != nil {
			return nil, fmt.Errorf("write chain: %w", err)
		}
	} else {
		// try to extract chain from Certificate bundle: certData.Cert may already include chain
		if err := os.WriteFile(chainPath, []byte(""), 0644); err != nil {
			// ignore
		}
	}

	c.log.Debug("Certificate files saved successfully")
	return &models.CertificatePaths{Cert: certPath, Key: keyPath, Chain: chainPath}, nil
}

func (c *Client) DeleteCertificateFiles(domain string) error {
	c.log.Info("Deleting certificate files for domain: ", domain)
	dir := filepath.Join(c.cfg.Certs.StorageDir, domain)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("certificate not found for domain: %s", domain)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove certificate directory: %w", err)
	}

	return nil
}

func (c *Client) loadOrCreatePrivateKey(path string) (crypto.PrivateKey, error) {
	c.log.Debug("loadOrCreatePrivateKey(): called, path=", path)
	if _, err := os.Stat(path); err == nil {
		// load
		c.log.Debug("Key file exists. Loading...")
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read key file: %w", err)
		}
		block, _ := pem.Decode(b)
		if block == nil {
			return nil, fmt.Errorf("invalid pem in key file")
		}
		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		return priv, nil
	}

	// create
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	b := x509.MarshalPKCS1PrivateKey(priv)
	pemBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}
	if err := os.WriteFile(path, pem.EncodeToMemory(pemBlock), 0600); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}
	c.log.Debug("Key loaded successfully")
	return priv, nil
}

func uniqueDomains(domains []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

type legoDNSWrapper struct {
	prov challenge.Provider
}

func (w *legoDNSWrapper) CreateTXTRecord(ctx context.Context, domain, name, value string, ttl int) error {
	return w.prov.Present(domain, name, value)
}

func (w *legoDNSWrapper) DeleteTXTRecord(ctx context.Context, domain, name, value string) error {
	return w.prov.CleanUp(domain, name, value)
}
