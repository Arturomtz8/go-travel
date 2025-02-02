resource "google_project_service" "artifact_registry" {
  project = var.project_id
  service = "artifactregistry.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy        = false
}

resource "google_artifact_registry_repository" "registry" {
  provider = google
  
  location      = var.region
  repository_id = var.repository_id
  description   = var.repository_description
  format        = "DOCKER"
  
  depends_on = [google_project_service.artifact_registry]
}

