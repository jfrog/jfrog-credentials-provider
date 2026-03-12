# JFrog Credential Provider Helm Chart - Changelog

All notable changes to this Helm chart will be documented in this file.

## [1.0.1] - 12th Mar, 2026
* Added KEP-4412 - Pod Level Identity Support For JFrog Artifactory on GCP

## [1.0.0] - 23rd Feb, 2026
* Allow using an existing ServiceAccount when `serviceAccount.create=false`
* Fixed `defaultCacheDuration` for AWS
* Updated timeout to 60 seconds for tailing for logs in init-container
* Added automatic rollback incase of config issues causing kubelet restarts
* **Breaking Change** 
* Moved `initContainer.image.imagePullSecrets` and `image.imagePullSecrets` to top-level `imagePullSecrets` to align with K8s spec.