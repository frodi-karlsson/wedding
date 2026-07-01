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
  type    = string
  default = "frodi.carla@gmail.com"
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

