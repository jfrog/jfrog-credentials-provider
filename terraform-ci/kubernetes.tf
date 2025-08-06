# Configure the Kubernetes provider
provider "kubernetes" {
  host                   =  var.create_eks_cluster ? module.eks[0].cluster_endpoint : data.aws_eks_cluster.eks_cluster_data.endpoint
  cluster_ca_certificate = var.create_eks_cluster ? base64decode(module.eks[0].cluster_certificate_authority_data) : (data.aws_eks_cluster.eks_cluster_data.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.eks_cluster_auth.token
}


resource "kubernetes_pod_v1" "busybox_pod_ds" {
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

resource "kubernetes_pod_v1" "busybox_pod" {
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
