# --- DigitalOcean ---
variable "do_token" {
  type        = string
  sensitive   = true
  description = "DigitalOcean API token."
}

variable "ssh_public_key" {
  type        = string
  description = "SSH public key to authorize on the droplet."
}

# --- Cloudflare ---
variable "cloudflare_api_token" {
  type        = string
  sensitive   = true
  description = "Cloudflare API token with Pages + DNS edit permissions."
}

variable "cloudflare_account_id" {
  type        = string
  sensitive   = true
  description = "Cloudflare account ID."
}

variable "cloudflare_zone_id" {
  type        = string
  sensitive   = true
  description = "Cloudflare zone ID for the domain (find in Cloudflare dashboard → domain → Overview → API section on the right)."
}

variable "ssh_allowed_ip" {
  type        = string
  description = "Your public IP CIDR (e.g. 84.55.97.207/32) for SSH access. If your IP changes, update this and re-apply. Break-glass: use the DO web console (https://cloud.digitalocean.com/droplets → Access → Launch Web Console)."
}

variable "r2_access_key_id" {
  type        = string
  sensitive   = true
  description = "Cloudflare R2 S3-compatible access key ID for backups. Create at Cloudflare → R2 → Manage R2 API Tokens."
}

variable "r2_secret_access_key" {
  type        = string
  sensitive   = true
  description = "Cloudflare R2 S3-compatible secret access key."
}

variable "r2_account_id" {
  type        = string
  sensitive   = true
  description = "Cloudflare account ID for R2 (same as cloudflare_account_id)."
}

variable "healthchecks_ping_url" {
  type        = string
  sensitive   = true
  description = "healthchecks.io ping URL for backup dead-man's-switch. Create a free check at healthchecks.io, paste its ping URL here. If a nightly ping is missed, healthchecks.io emails you."
}

# --- App secrets (written to droplet .env via cloud-init) ---
variable "resend_api_key" {
  type      = string
  sensitive = true
}

variable "admin_password" {
  type      = string
  sensitive = true
}

variable "session_secret" {
  type      = string
  sensitive = true
}

variable "resend_from" {
  type    = string
  default = "rsvp@carlaochfrodi.wedding"
}

variable "resend_to" {
  type        = string
  description = "Recipient for RSVP notifications and backup-failure alerts. Required (no default) so the address is set deliberately per deployment."
}

variable "cors_allowed_origins" {
  type    = string
  default = "https://carlaochfrodi.wedding"
}

# --- Domain / project ---
variable "domain" {
  type    = string
  default = "carlaochfrodi.wedding"
}

variable "pages_project_name" {
  type    = string
  default = "carlaochfrodi-wedding"
}

variable "do_region" {
  type    = string
  default = "ams3"
}

variable "droplet_size" {
  type    = string
  default = "s-1vcpu-1gb"
}

