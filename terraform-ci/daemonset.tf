# # Create a namespace for the JFrog resources

# resource "kubernetes_config_map" "jfrog_credential_provider_bootstrap" {
#   depends_on = [ kubernetes_namespace.jfrog_namespace ]
#   metadata {
#     name = "jfrog-credential-provider-bootstrap"
#     namespace = var.jfrog_namespace
#   }

#   data = {
#     "bootstrap.sh" = file("${path.module}/jfrog/k8s-bootstrap.sh")
#   }
# }

# resource "kubernetes_config_map" "jfrog_credential_provider_config" {
#   depends_on = [ kubernetes_namespace.jfrog_namespace ]
#   metadata {
#     name = "jfrog-credential-provider-config"
#     namespace = var.jfrog_namespace
#   }

#   data = {
#     "jfrog-provider.json" = <<-EOF
#     {
#       "name": "jfrog-credential-provider",
#       "matchImages": [
#         "*.jfrog.io"
#       ],
#       "defaultCacheDuration": "5h",
#       "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
#       "env": [
#         {
#           "name": "artifactory_url",
#           "value": "${var.artifactory_url}"
#         },
#         {
#           "name": "aws_auth_method",
#           "value": "assume_role"
#         },
#         {
#           "name": "aws_role_name",
#           "value": "${module.daemonset_test_ng.iam_role_name}"
#         }
#       ]
#     }
#     EOF
#   }
# }

# resource "kubernetes_daemonset" "jfrog_credential_provider" {
#   depends_on = [
#     kubernetes_namespace.jfrog_namespace,
#     kubernetes_config_map.jfrog_credential_provider_bootstrap,
#     kubernetes_config_map.jfrog_credential_provider_config,
#     module.daemonset_test_ng
#   ]

#   metadata {
#     name      = "jfrog-credential-provider-injector"
#     namespace = var.jfrog_namespace
#     labels = {
#       app = "jfrog-credential-provider"
#     }
#   }

#   spec {
#     selector {
#       match_labels = {
#         app = "jfrog-credential-provider"
#       }
#     }
#     template {
#       metadata {
#         labels = {
#           app = "jfrog-credential-provider"
#         }
#       }

#       spec {
#         host_pid = true
#         toleration {
#           key      = "jfrog-kubelet-daemonset-ng"
#           operator = "Equal"
#           value    = "true"
#           effect   = "NoSchedule"
#         }
#         node_selector = {
#           "createdBy": "kubelet-plugin-test-ci",
#           "nodeType": "daemonset"
#         }
#         # Init container to download the JFrog Credential Provider binary, update the kubelet configuration and restart the kubelet
#         init_container {
#           name = "jfrog-credential-provider-injector"
#           image = var.alpine_tools_image

#           env {
#             name = "JFROG_CREDENTIAL_PROVIDER_BINARY_URL"
#             value = var.jfrog_credential_provider_binary_url
#           }

#           command = [
#             "/bin/bash",
#             "-c",
#             ". /bin/bootstrap.sh"
#           ]

#           security_context {
#             privileged = true
#           }

#           volume_mount {
#             mount_path = "/host"
#             name       = "host"
#           }

#           volume_mount {
#             mount_path = "/bin/bootstrap.sh"
#             sub_path   = "bootstrap.sh"
#             name       = "jfrog-credential-provider-bootstrap"
#           }

#           volume_mount {
#             mount_path = "/etc/jfrog-provider.json"
#             sub_path   = "jfrog-provider.json"
#             name       = "jfrog-credential-provider-config"
#           }

#           resources {
#             limits = {
#               cpu    = "100m"
#               memory = "200Mi"
#             }
#             requests = {
#               cpu    = "5m"
#               memory = "10Mi"
#             }
#           }
#         }

#         # Pause container to keep the pod up with minimal resources until next host restart
#         container {
#           name  = "jfrog-credential-provider-injector-pause"
#           image = var.pause_image
#         }

#         volume {
#           name = "host"
#           host_path {
#             path = "/"
#             type = "Directory"
#           }
#         }

#         volume {
#           name = "jfrog-credential-provider-bootstrap"
#           config_map {
#             name = kubernetes_config_map.jfrog_credential_provider_bootstrap.metadata[0].name
#           }
#         }
#         volume {
#           name = "jfrog-credential-provider-config"
#           config_map {
#             name = kubernetes_config_map.jfrog_credential_provider_config.metadata[0].name
#           }
#         }
#         termination_grace_period_seconds = 5
#       }
#     }
#   }
# }

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
      key      = "jfrog-kubelet-daemonset-ng"
      operator = "Equal"
      value    = "true"
      effect   = "NoSchedule"
    }
    node_selector = {
      "createdBy": "kubelet-plugin-test-ci",
      "nodeType": "daemonset"
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
    module.manage_eks_nodes_using_jfrog_credential_plugin
  ]
}