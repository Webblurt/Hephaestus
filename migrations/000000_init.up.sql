-- Enable UUID generator
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- DOMAINS
-- ============================================================
CREATE TABLE IF NOT EXISTS domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_name VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending | active | renewal_failed | expired | deleted
    auto_renew BOOLEAN DEFAULT TRUE NOT NULL,
    verification_method VARCHAR(50) DEFAULT 'http-01',  -- http-01 | dns-01
    nginx_container_name TEXT NOT NULL,
    dns_provider VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT,
    deleted_at TIMESTAMPTZ,
    deleted_by TEXT,
    CHECK ((deleted_at IS NULL) = (deleted_by IS NULL))
);

COMMENT ON TABLE domains IS
    'Registered domains managed by the SSL manager. Each domain can have multiple certificates over time.';
COMMENT ON COLUMN domains.domain_name IS 'Fully qualified domain name (FQDN), e.g. example.com.';
COMMENT ON COLUMN domains.status IS 'Current domain state (pending, active, renewal_failed, etc.).';
COMMENT ON COLUMN domains.auto_renew IS 'If true, the system will automatically attempt certificate renewal.';
COMMENT ON COLUMN domains.verification_method IS 'ACME verification method used for domain validation.';

CREATE TABLE IF NOT EXISTS alternative_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE CASCADE,
    domain_name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT,
    deleted_at TIMESTAMPTZ,
    deleted_by TEXT,
    CHECK ((deleted_at IS NULL) = (deleted_by IS NULL))
);

COMMENT ON TABLE alternative_domains IS
    'Registered sub domain to its main.';
COMMENT ON COLUMN alternative_domains.domain_name IS 'Fully qualified domain name (FQDN), e.g. api.example.com.';

-- ============================================================
-- CERTIFICATES
-- ============================================================
CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE CASCADE,
    issuer VARCHAR(255) DEFAULT 'Let''s Encrypt',
    cert_path TEXT NOT NULL,
    key_path TEXT NOT NULL,
    chain_path TEXT,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    last_renewal TIMESTAMPTZ,
    renewal_attempts INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT,
    deleted_at TIMESTAMPTZ,
    deleted_by TEXT,
    CHECK ((deleted_at IS NULL) = (deleted_by IS NULL))
);

COMMENT ON TABLE certificates IS
    'SSL/TLS certificates issued for a given domain. Multiple certificates may exist due to renewals or reissues.';
COMMENT ON COLUMN certificates.cert_path IS 'Absolute path to the PEM-encoded certificate file.';
COMMENT ON COLUMN certificates.key_path IS 'Absolute path to the private key file.';
COMMENT ON COLUMN certificates.chain_path IS 'Path to the certificate chain file (optional).';
COMMENT ON COLUMN certificates.valid_from IS 'Certificate validity start date.';
COMMENT ON COLUMN certificates.valid_to IS 'Certificate expiry date.';
COMMENT ON COLUMN certificates.last_renewal IS 'Timestamp of the last successful renewal.';
COMMENT ON COLUMN certificates.renewal_attempts IS 'Number of renewal attempts made for this certificate.';


-- ============================================================
-- EVENTS / AUDIT LOG
-- ============================================================
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID REFERENCES domains(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,     -- created | renewed | failed | expired | deleted | manual_action
    message TEXT,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT,
    deleted_at TIMESTAMPTZ,
    deleted_by TEXT,
    CHECK ((deleted_at IS NULL) = (deleted_by IS NULL))
);

COMMENT ON TABLE events IS
    'Audit log of important events for domains and certificates.';
COMMENT ON COLUMN events.event_type IS 'Type of event (e.g. created, renewed, failed).';
COMMENT ON COLUMN events.message IS 'Human-readable description of what happened.';
COMMENT ON COLUMN events.metadata IS 'Optional structured data (e.g. error details, challenge info).';

-- ============================================================
-- INDEXES
-- ============================================================
CREATE INDEX idx_domains_domain_name ON domains(domain_name);
CREATE INDEX idx_domains_status ON domains(status);
CREATE INDEX idx_alternative_domains_domain_name ON alternative_domains(domain_name);
CREATE INDEX idx_certificates_valid_to ON certificates(valid_to);
CREATE INDEX idx_events_domain_id_created_at ON events(domain_id, created_at DESC);

-- ============================================================
-- TRIGGERS
-- ============================================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = statement_timestamp();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_domains_timestamp
BEFORE UPDATE ON domains
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_update_certificates_timestamp
BEFORE UPDATE ON certificates
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_update_events_timestamp
BEFORE UPDATE ON events
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
