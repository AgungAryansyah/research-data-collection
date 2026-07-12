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
make run
# Open http://localhost:8080
```

## Deployment

### 0. Setup (first time only)

```sh
# Beefy — install Go + create runtime directory
sudo apt install golang-go -y
sudo mkdir -p /opt/research-data-collection
sudo chown $USER /opt/research-data-collection

# VPS — install nginx + certbot, get SSL cert
sudo apt install nginx certbot python3-certbot-nginx -y
sudo certbot --nginx -d research.agungaryansyah.com
```

### 1. Beefy Machine

```sh
# Clone to your workspace (NOT into /opt/)
git clone <repo-url> ~/research-data-collection
cd ~/research-data-collection
make deploy-beefy TAILSCALE_IP=$(tailscale ip -4) DOMAIN=research.agungaryansyah.com
sudo -u research vim /opt/research-data-collection/config.json  # set admin credentials
```

### 2. VPS (nginx)

```sh
make deploy-nginx TAILSCALE_IP=<beefy-tailscale-ip> DOMAIN=research.agungaryansyah.com VPS_HOST=root@<vps-ip>
```

### 3. DNS

Add an A record: `research.agungaryansyah.com` → `<vps-public-ip>`

### 4. Verify

```sh
curl -s https://research.agungaryansyah.com/ | head -5
```

### Redeploy after code changes

```sh
make deploy-beefy TAILSCALE_IP=$(tailscale ip -4)  # on beefy machine
make deploy-nginx                                    # on dev machine (if nginx config changed)
```

## Makefile Targets

| Target              | Runs on | Purpose                                   |
| ------------------- | ------- | ----------------------------------------- |
| `make run`          | Dev     | Start local server                        |
| `make build`        | Any     | Compile binary                            |
| `make deploy-beefy` | Beefy   | Build + install binary + restart systemd  |
| `make deploy-nginx` | Dev     | Render nginx config + scp to VPS + reload |
| `make nginx-config` | Any     | Print rendered nginx config to stdout     |
| `make clean`        | Any     | Remove binary and `uploads/`              |

All deploy targets accept `TAILSCALE_IP`, `DOMAIN`, and (for nginx) `VPS_HOST` variables.

## Configuration

| Setting           | Where                                   | Default                    |
| ----------------- | --------------------------------------- | -------------------------- |
| Listen address    | `BIND_ADDR` env var                     | `:8080`                    |
| Storage path      | `config.json` → `storagePath`           | `uploads`                  |
| Video bitrate     | `config.json` → `videoBitrate`          | `5000000` (5 Mbps)         |
| Max resolution    | `config.json` → `maxWidth`/`maxHeight`  | `1920×1080`                |
| Chunk duration    | `config.json` → `chunkDurationMs`       | `500` ms                   |
| Admin credentials | `config.json` → `adminUser`/`adminPass` | `admin` / `admin`          |
| Form fields       | `config.json` → `infoFields`            | `["Name", "Age", "Notes"]` |
| Domain            | `deploy/nginx.conf`                     | —                          |
| Tailscale IP      | `deploy/nginx.conf` + systemd env       | —                          |

Most settings are editable from the admin dashboard at `/dashboard`.

## File Structure

```
uploads/<session-uuid>/
  info.json          # Participant form data
  metadata.json      # Camera label, resolution, frame rate, user agent
  takes.json         # Recording status per take
  video_take1.webm   # Recorded video
```
