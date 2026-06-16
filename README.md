# hover-dns

A lightweight CLI and DDNS daemon for managing DNS records on [Hover](https://www.hover.com) via their unofficial API.

> **Not affiliated with or endorsed by Hover.** This project uses an undocumented, unsupported API that may change or break at any time.

## What it does

- **List** DNS records across all your Hover domains
- **Set** any DNS record to a specific value
- **DDNS mode**: runs as a daemon, watches one or more record names, and automatically updates them to your current external IP on a configurable interval. Any non-A record (CNAME, TXT, MX, etc.) will be deleted and recreated as an A record on first update.
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
│   ├── set.go               # set subcommand
│   ├── add.go               # add subcommand
│   ├── delete.go            # delete subcommand
│   └── ddns.go              # ddns daemon subcommand
├── internal/
│   ├── hover/
│   │   ├── config.go        # Config struct and loader
│   │   ├── auth.go          # login / 2FA flow
│   │   ├── session.go       # session cookie persistence
│   │   ├── http.go          # HTTP client
│   │   ├── dns.go           # DNS record API calls
│   │   ├── totp.go          # TOTP code generation
│   │   ├── types.go         # API types
│   │   └── util.go          # name normalization helpers
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
