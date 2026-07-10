BINARY = research-data-collection
TAILSCALE_IP ?= 100.64.0.1
DOMAIN ?= research.agungaryansyah.com
VPS_HOST ?= root@<vps-ip>

.PHONY: build run clean deploy-beefy nginx-config deploy-nginx

build:
	go build -buildvcs=false -o $(BINARY) ./cmd/server

run:
	go run ./cmd/server

clean:
	rm -f $(BINARY)
	rm -rf uploads/

deploy-beefy: build
	@test -d /opt/research-data-collection || (echo "ERROR: /opt/research-data-collection not found. Create it first with: sudo mkdir -p /opt/research-data-collection && sudo chown $$USER /opt/research-data-collection"; exit 1)
	cp $(BINARY) /opt/research-data-collection/
	cp -r web /opt/research-data-collection/
	cp -n config.json /opt/research-data-collection/ 2>/dev/null || true
	sed 's|100.64.0.1|$(TAILSCALE_IP)|g' deploy/research-data.service | sudo tee /etc/systemd/system/research-data.service > /dev/null
	sudo useradd -r -s /bin/false research 2>/dev/null || true
	sudo chown -R research:research /opt/research-data-collection
	sudo systemctl daemon-reload
	sudo systemctl restart research-data
	@echo "Done. https://$(DOMAIN)"

nginx-config:
	@sed 's|<beefy-tailscale-ip>|$(TAILSCALE_IP)|g' deploy/nginx.conf | sed 's|research.yourdomain.com|$(DOMAIN)|g'

deploy-nginx: nginx-config
	@$(MAKE) --no-print-directory nginx-config > /tmp/nginx-research.conf
	scp /tmp/nginx-research.conf $(VPS_HOST):/tmp/
	ssh $(VPS_HOST) 'sudo cp /tmp/nginx-research.conf /etc/nginx/sites-available/research \
		&& sudo ln -sf /etc/nginx/sites-available/research /etc/nginx/sites-enabled/research \
		&& sudo rm -f /etc/nginx/sites-enabled/default \
		&& sudo nginx -t && sudo systemctl reload nginx'
	@rm -f /tmp/nginx-research.conf
	@echo "Done. https://$(DOMAIN)"
