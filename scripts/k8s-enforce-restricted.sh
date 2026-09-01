#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${1:-lyrahcle}"
CONTEXT="${2:-$(kubectl config current-context)}"

command -v kubectl >/dev/null 2>&1 || {
  printf 'missing required command: kubectl\n' >&2
  exit 1
}

kubectl --context "$CONTEXT" get namespace "$NAMESPACE" >/dev/null
kubectl --context "$CONTEXT" label namespace "$NAMESPACE" \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/enforce-version=latest \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/audit-version=latest \
  pod-security.kubernetes.io/warn=restricted \
  pod-security.kubernetes.io/warn-version=latest \
  --overwrite >/dev/null
kubectl --context "$CONTEXT" get namespace "$NAMESPACE" -o jsonpath='{.metadata.labels.pod-security\.kubernetes\.io/enforce}' | grep -Fxq restricted
printf 'pod security admission: restricted enforced on namespace %s\n' "$NAMESPACE"
