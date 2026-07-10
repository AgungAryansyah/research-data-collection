# Research Data Collection

A lightweight website for collecting webcam video recordings and camera metadata from research participants. Built in Go with vanilla JavaScript.

## Architecture

```
[Browser] --HTTPS--> [VPS: nginx] --Tailscale VPN--> [Beefy Machine: Go :8080]
                                                        |
                                                 [local disk: uploads/]
```

## Prerequisites

- Two Ubuntu machines connected via [Tailscale](https://tailscale.com)
- A domain name pointed at the VPS public IP
- Go 1.22+ on the beefy machine

## Quick Start (Local)

```sh
go run ./cmd/server
# Open http://localhost:8080
```

## Deployment

### 1. Beefy Machine

```sh
# Clone and build
git clone <repo-url> /opt/research-data-collection
cd /opt/research-data-collection
go build -buildvcs=false -o research-data-collection ./cmd/server

# Set up service user
sudo useradd -r -s /bin/false research
sudo chown -R research:research /opt/research-data-collection

# Set the Tailscale IP
TAILSCALE_IP=$(tailscale ip -4)
sudo sed -i "s|100.64.0.1|$TAILSCALE_IP|g" deploy/research-data.service

# Install and start
sudo cp deploy/research-data.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now research-data

# Wait for config.json to be auto-generated, then set credentials
sleep 2
sudo -u research nano /opt/research-data-collection/config.json
sudo systemctl restart research-data
```

### 2. VPS (nginx)

```sh
# Install nginx + certbot
sudo apt install nginx certbot python3-certbot-nginx -y

# Copy and configure the nginx template
scp deploy/nginx.conf user@vps:/tmp/nginx-research.conf
ssh user@vps

# Replace placeholders with real values
TAILSCALE_IP=<beefy-tailscale-ip>
sudo sed -i "s|<beefy-tailscale-ip>|$TAILSCALE_IP|g" /tmp/nginx-research.conf
sudo sed -i 's|research.yourdomain.com|research.agungaryansyah.com|g' /tmp/nginx-research.conf

# Install config
sudo cp /tmp/nginx-research.conf /etc/nginx/sites-available/research
sudo ln -s /etc/nginx/sites-available/research /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default

# Get SSL certificate
sudo certbot --nginx -d research.agungaryansyah.com

# Verify and reload
sudo nginx -t && sudo systemctl reload nginx
```

### 3. DNS

Add an A record: `research.agungaryansyah.com` → `<vps-public-ip>`

### 4. Verify

```sh
curl -s https://research.agungaryansyah.com/ | head -5
```

Should return the recording page HTML.

## Configuration

| Setting | Where | Default |
|---------|-------|---------|
| Listen address | `BIND_ADDR` env var | `:8080` |
| Storage path | `config.json` → `storagePath` | `uploads` |
| Video bitrate | `config.json` → `videoBitrate` | `5000000` (5 Mbps) |
| Max resolution | `config.json` → `maxWidth`/`maxHeight` | `1920×1080` |
| Chunk duration | `config.json` → `chunkDurationMs` | `500` ms |
| Admin credentials | `config.json` → `adminUser`/`adminPass` | `admin` / `admin` |
| Form fields | `config.json` → `infoFields` | `["Name", "Age", "Notes"]` |
| Domain | `deploy/nginx.conf` | — |
| Tailscale IP | `deploy/nginx.conf` + systemd env | — |

Most settings are editable from the admin dashboard at `/dashboard`.

## File Structure

```
uploads/<session-uuid>/
  info.json          # Participant form data
  metadata.json      # Camera label, resolution, frame rate, user agent
  takes.json         # Recording status per take
  video_take1.webm   # Recorded video
```
