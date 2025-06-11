provider "kubernetes" {
    config_path = var.kubeconfig_path
    // check if they are null 
    host = var.kubernetes_auth_object.host
    cluster_ca_certificate = var.kubernetes_auth_object.cluster_ca_certificate
    token = var.kubernetes_auth_object.token
}

resource "kubernetes_namespace" "jfrog_namespace" {
    count = var.jfrog_credential_plugin_daemonset_installation ? 1 : 0
    metadata {
        annotations = {
        name = var.daemonset_configuration.jfrog_namespace
        }

        labels = {
        app = "jfrog-credential-provider"
        }

        name = var.daemonset_configuration.jfrog_namespace
    }
}

resource "kubernetes_config_map" "jfrog_credential_provider_bootstrap" {

    count = var.jfrog_credential_plugin_daemonset_installation ? 1 : 0
    depends_on = [ kubernetes_namespace.jfrog_namespace ]
    metadata {
        name = "jfrog-credential-provider-bootstrap"
        namespace = var.daemonset_configuration.jfrog_namespace
    }

    data = {
        "bootstrap.sh" = file("${path.module}/jfrog/k8s-bootstrap.sh")
    }
}

resource "kubernetes_config_map" "jfrog_credential_provider_config" {

    count = var.jfrog_credential_plugin_daemonset_installation ? 1 : 0
    depends_on = [ kubernetes_namespace.jfrog_namespace, local_file.jfrog_provider_oidc, local_file.jfrog_provider_assume_role ]
    metadata {
        name = "jfrog-credential-provider-config"
        namespace = var.daemonset_configuration.jfrog_namespace
    }

    data = {
        "jfrog-provider.json" = local.jfrog_provider_config_content
    }
}

resource "kubernetes_daemonset" "jfrog_credential_provider" {

    count = var.jfrog_credential_plugin_daemonset_installation ? 1 : 0
    depends_on = [
        kubernetes_namespace.jfrog_namespace,
        kubernetes_config_map.jfrog_credential_provider_bootstrap,
        kubernetes_config_map.jfrog_credential_provider_config
    ]

    metadata {
        name      = "jfrog-credential-provider-injector"
        namespace = var.daemonset_configuration.jfrog_namespace
        labels = {
        app = "jfrog-credential-provider"
        }
    }

    spec {
        selector {
        match_labels = {
            app = "jfrog-credential-provider"
        }
        }
        template {
        metadata {
            labels = {
            app = "jfrog-credential-provider"
            }
            annotations = {
                config_change = sha1(jsonencode(merge(
                    kubernetes_config_map.jfrog_credential_provider_config[0].data,
                    kubernetes_config_map.jfrog_credential_provider_bootstrap[0].data
                )))
            }
        }

        spec {
            host_pid = true
            dynamic "toleration" {
            for_each = var.daemonset_configuration.tolerations
            content {
                key      = toleration.value.key
                operator = toleration.value.operator
                value    = toleration.value.value
                effect   = toleration.value.effect
                # toleration_seconds = toleration.value.toleration_seconds # Uncomment if using
            }
            }
            node_selector = {
            for pair in var.daemonset_configuration.node_selector : pair.key => pair.value
            }
            # Init container to download the JFrog Credential Provider binary, update the kubelet configuration and restart the kubelet
            init_container {
            name = "jfrog-credential-provider-injector"
            image = var.alpine_tools_image

            env {
                name = "JFROG_CREDENTIAL_PROVIDER_BINARY_URL"
                value = var.jfrog_credential_provider_binary_url
            }

            command = [
                "/bin/bash",
                "-c",
                ". /bin/bootstrap.sh"
            ]

            security_context {
                privileged = true
            }

            volume_mount {
                mount_path = "/host"
                name       = "host"
            }

            volume_mount {
                mount_path = "/bin/bootstrap.sh"
                sub_path   = "bootstrap.sh"
                name       = "jfrog-credential-provider-bootstrap"
            }

            volume_mount {
                mount_path = "/etc/jfrog-provider.json"
                sub_path   = "jfrog-provider.json"
                name       = "jfrog-credential-provider-config"
            }

            resources {
                limits = {
                cpu    = "100m"
                memory = "200Mi"
                }
                requests = {
                cpu    = "5m"
                memory = "10Mi"
                }
            }
            }

            # Pause container to keep the pod up with minimal resources until next host restart
            container {
            name  = "jfrog-credential-provider-injector-pause"
            image = var.pause_image
            }

            volume {
            name = "host"
            host_path {
                path = "/"
                type = "Directory"
            }
            }

            volume {
            name = "jfrog-credential-provider-bootstrap"
            config_map {
                name = kubernetes_config_map.jfrog_credential_provider_bootstrap[0].metadata[0].name
            }
            }
            volume {
            name = "jfrog-credential-provider-config"
            config_map {
                name = kubernetes_config_map.jfrog_credential_provider_config[0].metadata[0].name
            }
            }
            termination_grace_period_seconds = 5
        }
        }
    }
}