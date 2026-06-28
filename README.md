# hover-dns

A lightweight CLI and DDNS daemon for managing DNS records on [Hover](https://www.hover.com) via their unofficial API.

> **Not affiliated with or endorsed by Hover.** This project uses an undocumented, unsupported API that may change or break at any time.

## What it does

- **Manage records** from the CLI: list, add, set, and delete records across all your Hover domains
- **DDNS mode**: runs as a daemon, watches one or more record names, and automatically updates them to your current external IP on a configurable interval. Any non-A record (CNAME, TXT, MX, etc.) will be deleted and recreated as an A record on first update.
- **HTTP API**: `serve` runs the DDNS daemon together with a small, key-authenticated HTTP API for listing, creating, updating, and deleting records (see [HTTP API](#http-api-serve-mode))
- Generates TOTP codes from your 2FA secret automatically, no manual code entry

---

## Quickstart

### Install: Go install

```bash
go install github.com/bttnns/hover-dns@latest
```

### Install: Build from source

```bash
git clone https://github.com/bttnns/hover-dns.git
cd hover-dns
make build
```

### Install: Docker (DDNS daemon)

```bash
git clone https://github.com/bttnns/hover-dns.git
cd hover-dns
cp config.example.json config.json   # fill in credentials and record config
docker build -t hover-dns .
docker run -d --name hover-dns --restart unless-stopped \
  -v ./config.json:/config.json hover-dns ddns
docker logs -f hover-dns
```

### Setup

```bash
# Copy and fill in your config
cp config.example.json config.json

# Check your external IP (no config needed)
hover-dns ip

# List all DNS records
hover-dns list

# List records for one domain
hover-dns list example.com

# Update a record manually
hover-dns set example.com @ 1.2.3.4

# Run the DDNS daemon (domain, record_names, interval read from config.json)
hover-dns ddns
```

---

## Configuration

Copy `config.example.json` to `config.json` and fill in your values:

```json
{
  "username": "you@example.com",
  "password": "yourpassword",
  "totp_secret": "YOURBASE32SECRET",

  "domain": "example.com",
  "record_names": ["@", "home"],
  "interval": 46800
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `username` | always | Hover account email |
| `password` | always | Hover account password |
| `totp_secret` | if 2FA enabled | Base32 TOTP secret from your authenticator app |
| `domain` | ddns only | Domain name to watch (e.g. `example.com`) |
| `record_names` | ddns only | DNS hostnames to keep updated (`@` = apex, `home` = `home.example.com`) |
| `interval` | ddns only | Poll interval in seconds (default: 46800 = 13 hours) |

`domain`, `record_names`, and `interval` are only required for `ddns` mode. `totp_secret` is only required if your account has 2FA enabled.

After the first login, a session file (`.hover-dns.session`) is created next to your config and reused on subsequent commands, so no re-login is needed until the session expires. Add it to `.gitignore`.

To find your record names, run:

```bash
./hover-dns list example.com
```

Record names are in the NAME column (e.g. `@`, `home`, `www`).

---

## Commands

| Command | Description |
|---------|-------------|
| `hover-dns ip` | Print current external IP (no auth required) |
| `hover-dns list [domain]` | List DNS records for all domains, or filter by domain name/ID |
| `hover-dns set <domain> <name> <value>` | Update a DNS record's value |
| `hover-dns add <domain> <name> <type> <value>` | Add a new DNS record |
| `hover-dns delete <record-id>` | Delete a DNS record |
| `hover-dns ddns` | Run DDNS daemon, updating records to external IP (reads from config) |
| `hover-dns serve` | Run the DDNS daemon and an authenticated HTTP API together (see [HTTP API](#http-api-serve-mode)) |

Valid record types: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `SRV`

Global flags available on all commands:

| Flag | Description |
|------|-------------|
| `--config <path>` | Path to config file (default: `config.json`) |
| `-v`, `--verbose` | Verbose request/response logging |

Example:

```bash
./hover-dns --config /etc/hover-dns.json -v set example.com @ 1.2.3.4
```

---

## HTTP API (serve mode)

`hover-dns serve` runs the DDNS daemon and a small HTTP API in one process,
sharing a single authenticated Hover session. It is meant for a trusted network:
by default it binds loopback only, and every `/v1` request must present an API key.

```bash
export HOVER_API_KEY="$(openssl rand -hex 32)"   # required; serve refuses to start without it
hover-dns serve                                   # listens on 127.0.0.1:8088
hover-dns serve --listen 127.0.0.1:9000           # or set HOVER_LISTEN
```

### Authentication

Send the key as either header. `/healthz` is public; everything under `/v1`
returns `401` without a valid key.

```
Authorization: Bearer <key>
X-API-Key: <key>
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness check (no auth) |
| `GET` | `/v1/domains` | List all domains |
| `GET` | `/v1/domains/{domain}/records` | List records; optional `?name=` and `?type=` filters |
| `POST` | `/v1/domains/{domain}/records` | Create a record (`{"name","type","value"}`) |
| `PUT` | `/v1/domains/{domain}/records/{id}` | Update a record's value (and optionally `type`) |
| `DELETE` | `/v1/domains/{domain}/records/{id}` | Delete a record |

`{domain}` may be a domain name or its Hover id. Names are normalized: `@` is the
apex, and a short name is expanded to `name.domain`.

### Examples

```bash
KEY="$HOVER_API_KEY"
BASE="http://127.0.0.1:8088"

# list domains
curl -s -H "X-API-Key: $KEY" "$BASE/v1/domains"

# list A records for one domain
curl -s -H "X-API-Key: $KEY" "$BASE/v1/domains/example.com/records?type=A"

# create a record
curl -s -H "X-API-Key: $KEY" -X POST "$BASE/v1/domains/example.com/records" \
  -d '{"name":"www","type":"A","value":"203.0.113.10"}'

# delete a record by id
curl -s -H "X-API-Key: $KEY" -X DELETE "$BASE/v1/domains/example.com/records/<id>"
```

A record is returned as:

```json
{ "id": "dns1234567", "name": "www.example.com", "type": "A", "ttl": 900, "value": "203.0.113.10" }
```

Errors map to status codes: `400` invalid input, `401` missing/bad key, `404`
unknown domain/record, `429` Hover rate limit, `500` otherwise.

### Running behind a reverse proxy

`serve` binds loopback and speaks plain HTTP; it has no TLS of its own. For
access beyond the host, front it with a reverse proxy that terminates TLS and
restricts access to your network. Keep the API key secret and rotate it by
changing `HOVER_API_KEY` and restarting.

---

## Make targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary |
| `make install` | Build and install to `/usr/local/bin` |
| `make uninstall` | Remove from `/usr/local/bin` |
| `make clean` | Remove built binary |

---

## Docker

The Dockerfile is multi-stage and produces a minimal `scratch` image holding just the static binary and a CA bundle (no shell, no package manager).

```bash
# Build image
docker build -t hover-dns .

# One-off: list records
docker run --rm -v ./config.json:/config.json hover-dns list

# One-off: update a record
docker run --rm -v ./config.json:/config.json hover-dns set example.com @ 1.2.3.4

# Daemon: run ddns in the background
docker run -d --name hover-dns --restart unless-stopped \
  -v ./config.json:/config.json hover-dns ddns
docker logs -f hover-dns
```

---

## Project structure

```
hover-dns/
├── main.go                  # entry point
├── cmd/
│   ├── root.go              # cobra root command, --config and -v flags
│   ├── ip.go                # ip subcommand
│   ├── list.go              # list subcommand
│   ├── add.go               # add subcommand
│   ├── set.go               # set subcommand
│   ├── delete.go            # delete subcommand
│   ├── ddns.go              # ddns daemon subcommand
│   └── serve.go             # serve: ddns daemon + HTTP API
├── internal/
│   ├── hover/               # Hover client: auth, session, transport, DNS ops
│   │   ├── config.go        # Config struct and loader
│   │   ├── auth.go          # login / 2FA flow
│   │   ├── session.go       # session cookie persistence
│   │   ├── http.go          # HTTP transport + transparent re-login
│   │   ├── dns.go           # low-level (id-keyed) DNS record API calls
│   │   ├── ops.go           # high-level name-friendly operations (CLI + API)
│   │   ├── errors.go        # sentinel errors + HTTP status mapping
│   │   ├── totp.go          # TOTP code generation
│   │   ├── types.go         # API types
│   │   └── util.go          # name normalization helpers
│   ├── ddns/
│   │   └── ddns.go          # DDNS loop
│   ├── api/                 # internal HTTP API (chi): server, handlers, auth
│   │   ├── server.go
│   │   ├── handlers.go
│   │   └── auth.go
│   └── util/
│       └── ip.go            # external IP lookup
├── config.example.json
└── Dockerfile
```

---

## Notes

- Hover's API is unofficial and undocumented, so it may change without notice
- Record updates work via DELETE + POST (the PUT endpoint rejects updates)
- DDNS mode loads current DNS values on first iteration, then tracks state in memory
- Any non-A record watched by `ddns` (CNAME, TXT, MX, etc.) will be deleted and recreated as an A record on first IP mismatch
