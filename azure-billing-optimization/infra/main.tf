#==================================================================================
# ROOT MODULE - main.tf
#==================================================================================

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.75.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.45.0"
    }
  }
  
  backend "azurerm" {
    resource_group_name  = "rg-terraform-state"
    storage_account_name = "sttfstateproduction001"
    container_name       = "tfstate"
    key                  = "billing-optimization/terraform.tfstate"
  }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = true
    }
    key_vault {
      purge_soft_delete_on_destroy    = false
      recover_soft_deleted_key_vaults = true
    }
  }
}

# Data sources for existing resources
data "azurerm_client_config" "current" {}


#================================================================================
# Have skipped the rest implementation otherwise it wil be very long main pseudo 
# code related to solutions are present.
#================================================================================
