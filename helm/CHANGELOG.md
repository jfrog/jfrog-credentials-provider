# JFrog Credential Provider Helm Chart - Changelog

All notable changes to this Helm chart will be documented in this file.

## [0.1.0-beta.7] - 23rd Feb, 2026
* Allow using an existing ServiceAccount when `serviceAccount.create=false`
* **Breaking Change** 
* Moved `initContainer.image.imagePullSecrets` and `image.imagePullSecrets` to top-level `imagePullSecrets` to align with K8s spec.