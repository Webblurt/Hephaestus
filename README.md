# Hephaestus
An automated certificate service powered by ACME DNS-01.

Hephaestus is a lightweight service that creates and renews TLS certificates using the ACME DNS-01 challenge.

---

## Features

- Automatic certificate creation using ACME DNS-01
- Scheduled renewal (default: 30 days before expiration)
- TLS 1.3 ready
- PostgreSQL storage for domains and certificate metadata
- Simple and clean REST API
- Pluggable DNS providers
- File-based certificate output (`fullchain.pem`, `privkey.pem`)
- Lightweight Go codebase

---

## API Overview

API Base: `/hephaestus/api/v1`

### Routes

| Method | Endpoint | Description | Params |
|--------|----------|-------------|--------|
| `GET` | `/domains` | List all domains and certificate statuses | **in query** `status` - string, not required; `domain_name` - string, not required; `page_size` - int, not required; `page` - int, not required; |
| `POST` | `/domains` | Create a domain entry and automatically forge a certificate | **in body** `domain` - string, required; `nginx_container_name(your service working on)` - string, required; `dns_provider` - string, required; `alternative_domains` - []string, not required; `verification_method` - string, not required; `auto_renew` - bool, not required; |
| `DELETE` | `/domains` | Remove domain and its certificate files | **in query** `domain_id` - string, not required; `domain_name` - string, required; |

---

## High-Level Architecture

- **ACME module**

Handles certificate creation & renewal with lego or a custom ACME client

- **DNS module**

Creates `_acme-challenge.<domain>` TXT records for DNS-01

- **Storage module**

PostgreSQL tables for domains and metadata (status, expiration)

- **Certificate forge**

Produces keypairs and certificates

- **Scheduler**

Daily expiration checks and renewal triggers

- **REST API**

Allows programmatic interaction

---

## Setup

### 1. Clone the repository


```bash
git clone git@github.com:Webblurt/Hephaestus.git
cd Hephaestus
```


### 2. Create a dedicated session (optional but recommended)

If you want the service to continue running after closing SSH, create a tmux session:

```bash
tmux new -s forge

```


### 3. Configure environment variables for PostgreSQL

The project uses Docker for the database.

You need to export the essential environment variables before starting it:

```bash
export POSTGRES_USER="youruser"
export POSTGRES_PASSWORD="yourpassword"
export POSTGRES_DB="yourdb"
```


### 4. Start the database

You can start PostgreSQL using the provided Makefile commands:

**Start normally:**

```bash
make db-up
```

**Or start with a rebuild:**

```bash
make db-up-build
```

This launches a PostgreSQL container using the credentials you exported in the previous step.


### 5. Set environment variables for the application

Before running the API service, you must provide the required variables.

Mandatory variables include:

```bash
export CONFIG_PATH="/path/to/config.yaml"
export APP_NAME="hephaestus"
export APP_VERSION="1.0.0"

export DB_HOST="yourhost"
export DB_PORT="5432"
export DB_USER="$POSTGRES_USER"
export DB_PASSWORD="$POSTGRES_PASSWORD"

export AUTH_ACCESS_KEY="your_access_secret"
export AUTH_REFRESH_KEY="your_refresh_secret"

export CERT_RENEWAL_DURATION="24"   # how often scheduler will check if token expired
export SERVER_PORT="localip:8080"
export LOG_LEVEL="info"
```

**IMPORTANT: API Keys (API_KEY_`<NAME>`)**

Inside your config file, you may define several API endpoints:

```yaml
apis:
  - name: cloudflare
    url: https://api.cloudflare.com
  - name: hetzner
    url: https://dns.hetzner.com
```

For **each** API entry, the service automatically expects an environment variable:

```php-template
API_KEY_<UPPERCASE_NAME>
```

For example:

For `name`: `cloudflare`

You must set:

```bash
export API_KEY_CLOUDFLARE="your-cloudflare-token"
```

The service dynamically builds the environment variable name based on the API name from the config.

If any required key is missing, the service will not start and will report which variable is missing.


### 6. Create the YAML config

Create a YAML file (the one you referenced in `CONFIG_PATH`):

Example:

```yaml
app_name: "hephaestus-yourhostname"
version: "1.0.0"

apis:
  - name: hetzner
    url: https://api.hetzner.com/api/v1/records
  - name: cloudflare
    url: https://api.cloudflare.com/client/v4
  - name: aws
    url: https://route53.amazonaws.com/
  - name: digitalocean
    url: https://digitalocean.com
  - name: auth
    url: https://authexample.com

components: # must compare with api names
  hetzner_cli: "hetzner" 
  cloudflare_cli: "cloudflare"
  route_53_cli: "aws"
  digitalocean_cli: "doctl"
  auth_cli: "auth"

aws_config:
  access_key: ""
  secret_key: ""
  region: "us-east-1"

database:
  name: "name"
  host: ""
  port: 
  user: ""
  password: ""
  database: "db"
  migration_path: "../../"

auth:
  access_sec_key: ""
  refresh_sec_key: ""

certs:
  storage_dir: "./certs"
  email: "admin@example.com"
  renewal_duration: "24h"   # how often scheduler will check if token expired

server:
  port: "lockalip:8080"

logger:
  log_level: "info"
```

Then point to it:

```bash
export CONFIG_PATH="/absolute/path/to/config.yaml"
```


### 9. Close port 8080 to the outside

```bash
sudo ufw deny 8080
```


### 8. Run the service

From the project root:

```bash
make run
```


### 9. Connecting via SSH tunnel

```bash
ssh -N -L 8080:localip:8080 user@serverip
```

---

Now you can make requests to the service throug `http://localip:8080/hephaestus/api/v1/`