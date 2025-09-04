# Create an Azure AD application
resource "azuread_application_registration" "jfrog_credentials_provider_ad_app" {
  display_name = "${var.aks_cluster_name}-application"
}

# Create a service principal
resource "azuread_service_principal" "jfrog_credentials_provider_ad_sp" {
  client_id = azuread_application_registration.jfrog_credentials_provider_ad_app.client_id
}

resource "azuread_application_federated_identity_credential" "federated_identity_credential" {
  application_id = azuread_application_registration.jfrog_credentials_provider_ad_app.id
  display_name   = "${var.aks_cluster_name}-federated-identity"
  description    = "Deployments for jfrog-credentials-provider"
  audiences      = ["api://AzureADTokenExchange"]
  issuer         = "https://login.microsoftonline.com/${data.azuread_client_config.current[0].tenant_id}/v2.0"
  subject        = data.azurerm_kubernetes_cluster.k8s[0].kubelet_identity[0].object_id
}