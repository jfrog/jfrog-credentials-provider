# Configure the Kubernetes provider
provider "kubernetes" {
  # Multi-cloud configuration
  host = var.enable_aws ? (
    var.create_eks_cluster ? module.eks[0].cluster_endpoint : data.aws_eks_cluster.eks_cluster_data[0].endpoint
  ) : var.enable_azure ? (
    var.create_aks_cluster ? azurerm_kubernetes_cluster.k8s[0].kube_config[0].host : data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].host
  ) : null
  
  cluster_ca_certificate = var.enable_aws ? (
    var.create_eks_cluster ? base64decode(module.eks[0].cluster_certificate_authority_data) : base64decode(data.aws_eks_cluster.eks_cluster_data[0].certificate_authority.0.data)
  ) : var.enable_azure ? (
     var.create_aks_cluster ? base64decode(azurerm_kubernetes_cluster.k8s[0].kube_config[0].cluster_ca_certificate) : base64decode(data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].cluster_ca_certificate)
  ) : null
  
  # AWS uses token authentication
  token = var.enable_aws ? data.aws_eks_cluster_auth.eks_cluster_auth[0].token : null
  
  # Azure uses client certificate authentication
  client_certificate = var.enable_azure ? (var.create_aks_cluster ? base64decode(azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_certificate) : base64decode(data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_certificate)) : null
  client_key         = var.enable_azure ? (var.create_aks_cluster ? base64decode(azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_key) : base64decode(data.azurerm_kubernetes_cluster.k8s[0].kube_config[0].client_key)) : null
}


# AWS busybox test pod for DaemonSet testing
resource "kubernetes_pod_v1" "busybox_pod_ds" {
  count = var.enable_aws ? 1 : 0
  
  metadata {
    name = "busybox-pod-ds-v1"
    labels = {
      app = "busybox"
    }
    namespace = var.jfrog_namespace
  }

  spec {
    toleration {
        key    = "forDaemonset"
        operator = "Equal"
        value  = "true"
        effect = "NoSchedule"
    }
    node_selector = {
      "onlyForDaemonset": "true"
    }
    container {
      name  = "busybox-container"
      # TODO Make this dynamic
      image = var.busybox_image_ds
      command = [
        "/bin/sh",
        "-c",
        "while true; do sleep 3600; done"
      ]
    }
  }

  depends_on = [
    module.create_daemonset_with_plugin_enabled
  ]
}

# AWS busybox test pod for node group testing
resource "kubernetes_pod_v1" "busybox_pod" {
  count = var.enable_aws ? 1 : 0
  
  metadata {
    name = "busybox-pod"
    labels = {
      app = "busybox"
    }
    namespace = var.jfrog_namespace
  }

  spec {
    toleration {
      key      = "jfrog-kubelet-oidc-ng"
      operator = "Equal"
      value    = "true"
      effect   = "NoSchedule"
    }
    node_selector = {
      "createdBy": "kubelet-plugin-test-ci",
      "nodeType": "cognito-oidc"
    }
    container {
      name  = "busybox-container"
      # TODO Make this dynamic
      image = "partnership-docker-remote-test.jfrog.io/busybox:latest"
      command = [
        "/bin/sh",
        "-c",
        "while true; do sleep 3600; done"
      ]
    }
  }

  depends_on = [module.create_daemonset_with_plugin_enabled]
}

resource "kubernetes_service_account_v1" "busybox_sa_wi" {
  metadata {
    name      = "busybox-sa-wi"
    namespace = var.jfrog_namespace
    annotations = length(aws_iam_role.eks_node_role) > 0 ? {
      "JFrogExchange" = "true"
      "eks.amazonaws.com/role-arn" = aws_iam_role.eks_node_role[0].arn
    } : {
      "JFrogExchange" = "true"
    }
  }
  depends_on = [
    aws_iam_role.eks_node_role,
    null_resource.update_eks_node_role_web_identity,
    module.manage_eks_nodes_using_jfrog_credential_plugin_web_identity
  ]
}

# AWS busybox test pod for node group testing with web identity
resource "kubernetes_pod_v1" "busybox_pod_wi" {
  count = var.enable_aws ? 1 : 0

  metadata {
    name = "busybox-pod-wi"
    labels = {
      app = "busybox"
    }
    namespace = var.jfrog_namespace
  }

  spec {
    service_account_name = kubernetes_service_account_v1.busybox_sa_wi.metadata[0].name
    toleration {
      key      = "jfrog-kubelet-oidc-ng"
      operator = "Equal"
      value    = "true"
      effect   = "NoSchedule"
    }
    node_selector = {
      "createdBy": "kubelet-plugin-test-ci",
      "nodeType": "web-identity"
    }
    container {
      name  = "busybox-container"
      # TODO Make this dynamic
      image = "partnership-docker-remote-test.jfrog.io/busybox:latest"
      command = [
        "/bin/sh",
        "-c",
        "while true; do sleep 3600; done"
      ]
    }
  }

  depends_on = [
    module.create_daemonset_with_plugin_enabled,
    kubernetes_service_account_v1.busybox_sa_wi
  ]
}

# Azure busybox test pod for DaemonSet testing
resource "kubernetes_pod_v1" "azure_busybox_pod_ds" {
  count = var.enable_azure ? 1 : 0
  
  metadata {
    name = "azure-busybox-pod-ds-v1"
    labels = {
      app = "busybox"
      cloud = "azure"
    }
    namespace = var.jfrog_namespace
  }

  spec {
    node_selector = {
      "azure-jfrog-test": "true"
    }
    container {
      name  = "busybox-container"
      image = var.busybox_image_ds
      command = [
        "/bin/sh",
        "-c",
        "while true; do sleep 3600; done"
      ]
    }
  }

  depends_on = [
    module.create_azure_daemonset_with_plugin_enabled
  ]
}
