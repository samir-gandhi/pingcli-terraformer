terraform {
  required_version = ">= 1.3"

  required_providers {
    pingone = {
      source  = "pingidentity/pingone"
      version = ">= 1.0.0"
    }
  }
}

provider "pingone" {
  # Provider configuration would go here in actual usage
  # For testing, we just need the provider block to be present
}
