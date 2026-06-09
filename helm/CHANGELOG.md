# JFrog Credential Provider Helm Chart - Changelog

All notable changes to this Helm chart will be documented in this file.

## [1.3.0] = 10th June, 2026
* Added support for `imds_direct` auth flow for Azure to get ride of federated credentials limit
* Updated projected token flow for Azure to remove app registeration

## [1.2.0] - 5th June, 2026
* Added a fix to honor proxy env vars for provider HTTP clients
* Add assume_external_role auth method for cross-account IAM role assumption

## [1.1.2] - 22nd May, 2026
* Added support for `containerLogging` for plugins logs in container stdout

## [1.1.1] - 21st April, 2026
* Added support for Azure China configured using `azure_cloud_name`

## [1.1.0] - 7th April, 2026
* Added KEP-4412 - Pod Level Identity Support For JFrog Artifactory on GCP
* Added support for `http_timeout_seconds` for HTTP calls
* Fixed `secret_ttl_seconds` in configmap to handle quotes
* Removed `host` header from AWS Signed requests to Artifactory to prevent from overriding host issues on webserver

## [1.0.1] - 25th Mar, 2026
* Added support for disabling auto-upgrade of binary through `autoUpgrade`
* Added support for `aws_region` for `assume_role` authentication method

## [1.0.0] - 23rd Feb, 2026
* Allow using an existing ServiceAccount when `serviceAccount.create=false`
* Fixed `defaultCacheDuration` for AWS
* Updated timeout to 60 seconds for tailing for logs in init-container
* Added automatic rollback incase of config issues causing kubelet restarts
* **Breaking Change** 
* Moved `initContainer.image.imagePullSecrets` and `image.imagePullSecrets` to top-level `imagePullSecrets` to align with K8s spec.
