# Generate SSH key for AKS nodes
resource "azapi_resource_action" "ssh_public_key_gen" {
  count = var.enable_azure && var.create_aks_cluster ? 1 : 0
  
  type        = "Microsoft.Compute/sshPublicKeys@2022-11-01"
  resource_id = azapi_resource.ssh_public_key[0].id
  action      = "generateKeyPair"
  method      = "POST"

  response_export_values = ["publicKey", "privateKey"]
}

resource "azapi_resource" "ssh_public_key" {
  count = var.enable_azure && var.create_aks_cluster ? 1 : 0
  
  type     = "Microsoft.Compute/sshPublicKeys@2022-11-01"
  name     = "jfrog-ssh-key-${var.azure_location}"
  location = var.azure_location
  parent_id = data.azurerm_resource_group.existing[0].id
}

# Data source for existing resource group
data "azurerm_resource_group" "existing" {
  count = var.enable_azure ? 1 : 0
  name  = var.azure_resource_group_name
}

# AKS Cluster
resource "azurerm_kubernetes_cluster" "k8s" {
  count = var.enable_azure && var.create_aks_cluster ? 1 : 0

  location            = var.azure_location
  name                = var.aks_cluster_name
  resource_group_name = var.azure_resource_group_name
  dns_prefix          = "${var.aks_cluster_name}-dns"
  
  oidc_issuer_enabled       = true
  workload_identity_enabled = true
  azure_policy_enabled      = true

  identity {
    type = "SystemAssigned"
  }

  default_node_pool {
    name       = "agentpool"
    vm_size    = var.azure_node_vm_size
    node_count = var.azure_node_count
    
    node_labels = {
      "azure-jfrog-test" = "true"
    }
  }

  linux_profile {
    admin_username = var.azure_admin_username

    ssh_key {
      key_data = azapi_resource_action.ssh_public_key_gen[0].output.publicKey
    }
  }

  network_profile {
    network_plugin    = "kubenet"
    load_balancer_sku = "standard"
  }

  api_server_access_profile {
    authorized_ip_ranges = var.azure_cluster_public_access_cidrs
  }

  tags = {
    Environment = "jfrog-ci"
    Purpose     = "credential-provider-testing"
  }
}

# Data source for AKS cluster (works for both existing and created clusters)
data "azurerm_kubernetes_cluster" "k8s" {
  count = var.enable_azure ? 1 : 0
  
  name                = var.create_aks_cluster ? azurerm_kubernetes_cluster.k8s[0].name : var.aks_cluster_name
  resource_group_name = var.azure_resource_group_name

  depends_on = [azurerm_kubernetes_cluster.k8s]
}