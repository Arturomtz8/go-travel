variable "project_id" {
  description = "The ID of the GCP project"
  type        = string
}

variable "region" {
  description = "The region where the resources will be created"
  type        = string
  default     = "us-west1"
}

variable "repository_id" {
  description = "The ID of the Artifact Registry repository"
  type        = string
}

variable "repository_description" {
  description = "The description of the Artifact Registry repository"
  type        = string
  default     = "Docker repository of container images for go-travel project"
}
