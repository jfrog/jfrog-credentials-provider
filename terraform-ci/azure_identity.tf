# Data source for existing Azure AD application (when using existing app)
data "azuread_application" "existing_jfrog_credentials_provider_ad_app" {
  count     = var.enable_azure && var.existing_azure_app_client_id != null ? 1 : 0
  client_id = var.existing_azure_app_client_id
}

# Data source for existing service principal (when using existing app)
data "azuread_service_principal" "existing_jfrog_credentials_provider_ad_sp" {
  count     = var.enable_azure && var.existing_azure_app_client_id != null ? 1 : 0
  client_id = var.existing_azure_app_client_id
}

# Create an Azure AD application (only when not using existing app)
resource "azuread_application_registration" "jfrog_credentials_provider_ad_app" {
  count = var.enable_azure && var.existing_azure_app_client_id == null ? 1 : 0
  display_name = "${var.aks_cluster_name}-application"
}

# Create a service principal (only when not using existing app)
resource "azuread_service_principal" "jfrog_credentials_provider_ad_sp" {
  count = var.enable_azure && var.existing_azure_app_client_id == null ? 1 : 0
  client_id = azuread_application_registration.jfrog_credentials_provider_ad_app[0].client_id
}

# Local values to reference the correct application and service principal
locals {
  azure_app_id = var.enable_azure ? (
    var.existing_azure_app_client_id != null ? 
    data.azuread_application.existing_jfrog_credentials_provider_ad_app[0].id : 
    azuread_application_registration.jfrog_credentials_provider_ad_app[0].id
  ) : null
  
  azure_client_id = var.enable_azure ? (
    var.existing_azure_app_client_id != null ? 
    var.existing_azure_app_client_id : 
    azuread_application_registration.jfrog_credentials_provider_ad_app[0].client_id
  ) : null
}

resource "azuread_application_federated_identity_credential" "federated_identity_credential" {
  count = var.enable_azure ? 1 : 0
  application_id = local.azure_app_id
  display_name   = "${var.aks_cluster_name}-federated-identity"
  description    = "Deployments for jfrog-credentials-provider"
  audiences      = ["api://AzureADTokenExchange"]
  issuer         = "https://login.microsoftonline.com/${data.azuread_client_config.current[0].tenant_id}/v2.0"
  subject        = data.azurerm_kubernetes_cluster.k8s[0].kubelet_identity[0].object_id

  lifecycle {
    create_before_destroy = true
  }
}