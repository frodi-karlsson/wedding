# --- SSH key ---
resource "digitalocean_ssh_key" "default" {
  name       = "wedding-deploy-key"
  public_key = var.ssh_public_key
}

# --- Container registry ---
# DO registry names are globally unique across ALL accounts. Use a specific name.
resource "digitalocean_container_registry" "wedding" {
  name                   = "carlaochfrodi"
  region                 = var.do_region
  subscription_tier_slug = "starter"
}

# --- Droplet ---
resource "digitalocean_droplet" "wedding" {
  name     = "wedding-backend"
  region   = var.do_region
  size     = var.droplet_size
  image    = "ubuntu-24-04-x64"
  ssh_keys = [digitalocean_ssh_key.default.fingerprint]
  user_data = templatefile("${path.module}/cloud-init.yaml.tftpl", {
    do_token             = var.do_token
    resend_api_key       = var.resend_api_key
    admin_password       = var.admin_password
    session_secret       = var.session_secret
    resend_from          = var.resend_from
    resend_to            = var.resend_to
    cors_allowed_origins = var.cors_allowed_origins
    domain               = var.domain
    registry_server      = digitalocean_container_registry.wedding.endpoint
    registry_server_url  = digitalocean_container_registry.wedding.server_url
  })
}

# --- Reserved IP (stable across droplet recreation) ---
resource "digitalocean_reserved_ip" "wedding" {
  region     = var.do_region
  droplet_id = digitalocean_droplet.wedding.id
}

# --- Volume for SQLite persistence ---
resource "digitalocean_volume" "data" {
  region                  = var.do_region
  name                    = "wedding-data"
  size                    = 1
  initial_filesystem_type = "ext4"
  description             = "SQLite data volume for wedding backend"
}

resource "digitalocean_volume_attachment" "data" {
  droplet_id = digitalocean_droplet.wedding.id
  volume_id  = digitalocean_volume.data.id
}

# --- Firewall ---
resource "digitalocean_firewall" "wedding" {
  name        = "wedding-firewall"
  droplet_ids = [digitalocean_droplet.wedding.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "all"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "all"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

# --- Cloudflare Pages project (direct upload, no git connection) ---
resource "cloudflare_pages_project" "wedding" {
  account_id        = var.cloudflare_account_id
  name              = var.pages_project_name
  production_branch = "main"

  deployment_configs = {
    preview = {
      env_vars = {
        PUBLIC_API_URL = {
          type  = "plain_text"
          value = "https://api.${var.domain}"
        }
      }
    }
    production = {
      env_vars = {
        PUBLIC_API_URL = {
          type  = "plain_text"
          value = "https://api.${var.domain}"
        }
      }
    }
  }
}

# --- DNS: apex CNAME → pages.dev (Cloudflare Pages custom domain) ---
resource "cloudflare_dns_record" "apex" {
  zone_id = var.cloudflare_zone_id
  name    = "@"
  content = "${var.pages_project_name}.pages.dev"
  type    = "CNAME"
  proxied = true
  ttl     = 1

  depends_on = [cloudflare_pages_project.wedding]
}

# --- Bind custom domain (apex) to the Pages project ---
resource "cloudflare_pages_domain" "main" {
  account_id   = var.cloudflare_account_id
  project_name = cloudflare_pages_project.wedding.name
  name         = var.domain

  depends_on = [cloudflare_dns_record.apex]
}

# --- DNS: api. subdomain → droplet (DNS-only so Caddy does TLS) ---
resource "cloudflare_dns_record" "api" {
  zone_id = var.cloudflare_zone_id
  name    = "api"
  content = digitalocean_reserved_ip.wedding.ip_address
  type    = "A"
  proxied = false
  ttl     = 1
}
