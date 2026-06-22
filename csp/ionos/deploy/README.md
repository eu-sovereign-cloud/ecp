This directory contains convenience scripts and templates to install Crossplane and the IONOS Crossplane provider for local development.

Prerequisites
- kubectl configured to talk to your cluster
- helm (for installing Crossplane)
- (optional) kubectl-crossplane plugin for `kubectl crossplane` commands

Quick start
1. Install Crossplane using Helm:

   ./install-crossplane.sh

2. Install the provider (replace PROVIDER_PKG with the provider package to install):

   PROVIDER_PKG="ionos-cloud/provider-upjet-ionoscloud:v0.6.0" ./install-provider.sh

3. Create credentials secret in `crossplane-system` and apply a ProviderConfig

   make create-secret
   make provider-config

Notes
- Provider installation varies between providers. The script `install-provider.sh` uses `kubectl crossplane install provider` if available. If that is not available or supported for the target provider, follow the provider's README for installation (helm or apply manifests).
- The `providerconfig-example.yaml` is a template — providers define their own ProviderConfig/ClusterProviderConfig CRDs and fields; fill it according to the provider documentation.

Files
- install-crossplane.sh — Helm-based Crossplane installer
- install-provider.sh  — Wrapper to install a Crossplane provider package
- providerconfig-example.yaml — Template for a provider credentials/configuration object
- Makefile — convenience targets
