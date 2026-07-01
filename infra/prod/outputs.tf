output "droplet_public_ip" {
  value = digitalocean_droplet.wedding.ipv4_address
}

output "reserved_ip" {
  value = digitalocean_reserved_ip.wedding.ip_address
}

output "registry_endpoint" {
  value     = digitalocean_container_registry.wedding.endpoint
  sensitive = true
}

output "pages_domain" {
  value = cloudflare_pages_domain.main.name
}
