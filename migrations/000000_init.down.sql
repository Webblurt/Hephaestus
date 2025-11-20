DROP TRIGGER IF EXISTS trg_update_domains_timestamp ON domains;
DROP TRIGGER IF EXISTS trg_update_certificates_timestamp ON certificates;
DROP TRIGGER IF EXISTS trg_update_acme_accounts_timestamp ON acme_accounts;
DROP TRIGGER IF EXISTS trg_update_events_timestamp ON events;

DROP FUNCTION IF EXISTS set_updated_at();

DROP INDEX IF EXISTS idx_domains_domain_name;
DROP INDEX IF EXISTS idx_domains_status;
DROP INDEX IF EXISTS idx_alternative_domains_domain_name;
DROP INDEX IF EXISTS idx_certificates_valid_to;
DROP INDEX IF EXISTS idx_events_domain_id_created_at;

DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS certificates CASCADE;
DROP TABLE IF EXISTS alternative_domains CASCADE;
DROP TABLE IF EXISTS domains CASCADE;

DROP EXTENSION IF EXISTS "pgcrypto";

