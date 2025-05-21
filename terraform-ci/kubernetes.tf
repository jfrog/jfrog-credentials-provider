# Configure the Kubernetes provider
provider "kubernetes" {
  host                   = data.aws_eks_cluster.self_managed_eks_cluster_data.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.self_managed_eks_cluster_data.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.self_managed_eks_cluster_auth.token
}

resource "kubernetes_namespace" "jfrog_namespace" {

  metadata {
    annotations = {
      name = var.jfrog_namespace
    }

    labels = {
      app = "jfrog-credential-provider"
    }

    name = var.jfrog_namespace
  }
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
      "nodeType": "cogito-oidc"
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

  depends_on = [kubernetes_namespace.jfrog_namespace]
}
