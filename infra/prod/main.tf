# --- SSH key ---
resource "digitalocean_ssh_key" "default" {
  name       = "wedding-deploy-key"
  public_key = var.ssh_public_key
}

# --- Container registry ---
resource "digitalocean_container_registry" "wedding" {
  name                   = "wedding"
  region                 = var.do_region
  subscription_tier_slug = "starter"
}

# --- Droplet ---
resource "digitalocean_droplet" "wedding" {
  name     = "wedding-backend"
  region   = var.do_region
  size     = var.droplet_size
  image    = "docker-20-04" # Docker preinstalled; verify this slug exists, else use "ubuntu-24-04" and install docker via cloud-init
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
    spaces_key           = var.backup_spaces_key
    spaces_secret        = var.backup_spaces_secret
    backup_bucket        = digitalocean_spaces_bucket.backups.name
    backup_region        = var.do_region
  })
}

# --- Reserved IP (stable across droplet recreation) ---
resource "digitalocean_reserved_ip" "wedding" {
  droplet_id = digitalocean_droplet.wedding.id
}

# --- Volume for SQLite persistence ---
resource "digitalocean_volume" "data" {
  name               = "wedding-data"
  region             = var.do_region
  size_gigabytes     = 1
  initial_filesystem = "ext4"
  description        = "SQLite data volume for wedding backend"
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

# --- Spaces bucket for backups ---
resource "digitalocean_spaces_bucket" "backups" {
  name   = "wedding-backups"
  region = var.do_region
  acl    = "private"
}

resource "digitalocean_spaces_bucket_lifecycle" "backups" {
  bucket = digitalocean_spaces_bucket.backups.name
  region = digitalocean_spaces_bucket.backups.region

  rule {
    id      = "expire-old-backups"
    enabled = true

    expiration {
      days = 30
    }
  }
}
