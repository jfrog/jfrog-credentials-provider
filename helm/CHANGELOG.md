# JFrog Credential Provider Helm Chart - Changelog

All notable changes to this Helm chart will be documented in this file.

## [1.0.1] - 28th Feb, 2026
* Added support for disabling auto-upgrade of binary through `autoUpgrade`

## [1.0.0] - 23rd Feb, 2026
* Allow using an existing ServiceAccount when `serviceAccount.create=false`
* Fixed `defaultCacheDuration` for AWS
* Updated timeout to 60 seconds for tailing for logs in init-container
* Added automatic rollback incase of config issues causing kubelet restarts
* **Breaking Change** 
* Moved `initContainer.image.imagePullSecrets` and `image.imagePullSecrets` to top-level `imagePullSecrets` to align with K8s spec.
