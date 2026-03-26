#!/usr/bin/env bash
set -euo pipefail

NAMESPACE=${NAMESPACE:-crossplane-system}
RELEASE=${RELEASE:-crossplane}
REPO_NAME=${REPO_NAME:-crossplane-stable}
REPO_URL=${REPO_URL:-https://charts.crossplane.io/stable}

# KUBECONFIG-only targeting for helm/kubectl
HELM_ARGS=${HELM_ARGS:-}
KUBECTL_ARGS=${KUBECTL_ARGS:-}
if [[ -n "${KUBECONFIG:-}" ]]; then
  HELM_ARGS+=" --kubeconfig ${KUBECONFIG}"
  KUBECTL_ARGS+=" --kubeconfig ${KUBECONFIG}"
fi

echo "Ensuring namespace ${NAMESPACE} exists"
kubectl ${KUBECTL_ARGS} create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl ${KUBECTL_ARGS} apply -f -

echo "Adding helm repo ${REPO_NAME} -> ${REPO_URL} (if not already present)"
helm ${HELM_ARGS} repo add ${REPO_NAME} ${REPO_URL} >/dev/null 2>&1 || true
helm ${HELM_ARGS} repo update

# If the release is already present in the namespace, upgrade; otherwise install.
if helm ${HELM_ARGS} ls -n "${NAMESPACE}" --filter "^${RELEASE}$" -q | grep -q "^${RELEASE}$"; then
  echo "Release ${RELEASE} already present in namespace ${NAMESPACE}; performing upgrade"
  helm ${HELM_ARGS} upgrade ${RELEASE} ${REPO_NAME}/crossplane --namespace ${NAMESPACE}
else
  echo "Installing Crossplane (release: ${RELEASE}) into namespace ${NAMESPACE}"
  helm ${HELM_ARGS} install ${RELEASE} --namespace ${NAMESPACE} ${REPO_NAME}/crossplane
fi

# Wait for CRDs and core controller to be ready
CRD_NAME="providers.pkg.crossplane.io"
echo "Waiting for CRD ${CRD_NAME} to be established..."
kubectl ${KUBECTL_ARGS} wait --for=condition=Established crd/${CRD_NAME} --timeout=120s >/dev/null 2>&1 || true

echo "Waiting for Crossplane deployment to be ready..."
kubectl ${KUBECTL_ARGS} -n ${NAMESPACE} rollout status deploy/${RELEASE} --timeout=180s || true

echo "Crossplane installation step finished"
