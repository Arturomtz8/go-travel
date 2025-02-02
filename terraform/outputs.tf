output "repository_id" {
  value       = google_artifact_registry_repository.registry.repository_id
  description = "The ID of the created repository"
}

output "registry_url" {
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${var.repository_id}"
  description = "The URL of the Artifact Registry repository"
}