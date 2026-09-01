#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER="${LYRAHCLE_KIND_CLUSTER:-lyrahcle}"
NAMESPACE="${LYRAHCLE_K8S_NAMESPACE:-lyrahcle}"
CONTEXT="kind-${CLUSTER}"
BUILD_IMAGES="${LYRAHCLE_K8S_BUILD_IMAGES:-1}"
RUN_SMOKE="${LYRAHCLE_K8S_RUN_SMOKE:-1}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$1" >&2
    exit 1
  }
}

for command in docker kind kubectl helm python3; do
  need "$command"
done

if ! kind get clusters 2>/dev/null | grep -Fxq "$CLUSTER"; then
  kind create cluster --name "$CLUSTER" --config "$ROOT/deploy/kind/lyrahcle.yaml"
fi

kubectl --context "$CONTEXT" create namespace "$NAMESPACE" --dry-run=client -o yaml | \
  kubectl --context "$CONTEXT" apply -f - >/dev/null
kubectl --context "$CONTEXT" label namespace "$NAMESPACE" \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/audit-version=latest \
  pod-security.kubernetes.io/warn=restricted \
  pod-security.kubernetes.io/warn-version=latest \
  --overwrite >/dev/null

if ! kubectl --context "$CONTEXT" -n "$NAMESPACE" get secret lyrahcle-local-datastore >/dev/null 2>&1; then
  kubectl --context "$CONTEXT" -n "$NAMESPACE" create secret generic lyrahcle-local-datastore \
    --from-literal=POSTGRES_USER=lyrahcle \
    --from-literal=POSTGRES_PASSWORD=lyrahcle \
    --from-literal=POSTGRES_DB=lyrahcle >/dev/null
fi

if ! kubectl --context "$CONTEXT" -n "$NAMESPACE" get secret lyrahcle-app-secrets >/dev/null 2>&1; then
  jwt_secret="${LYRAHCLE_LOCAL_JWT_SECRET:-$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(48))
PY
)}"
  admin_password="${LYRAHCLE_LOCAL_ADMIN_PASSWORD:-local-admin-change-me}"
  kubectl --context "$CONTEXT" -n "$NAMESPACE" create secret generic lyrahcle-app-secrets \
    --from-literal=DATABASE_URL='postgres://lyrahcle:lyrahcle@postgres:5432/lyrahcle?sslmode=disable' \
    --from-literal=REDIS_URL='redis:6379' \
    --from-literal=JWT_SECRET="$jwt_secret" \
    --from-literal=SEED_ADMIN_PASSWORD="$admin_password" >/dev/null
fi

kubectl --context "$CONTEXT" -n "$NAMESPACE" apply -f "$ROOT/deploy/k8s/local/dependencies.yaml"
kubectl --context "$CONTEXT" -n "$NAMESPACE" rollout status deployment/postgres --timeout=180s
kubectl --context "$CONTEXT" -n "$NAMESPACE" rollout status deployment/redis --timeout=180s

if [[ "$BUILD_IMAGES" == "1" ]]; then
  docker build -t lyrahcle/backend:dev -f "$ROOT/deploy/docker/Dockerfile.backend" "$ROOT"
  docker build -t lyrahcle/frontend:dev -f "$ROOT/deploy/docker/Dockerfile.frontend" "$ROOT"
  docker build -t lyrahcle/cve-sync:dev -f "$ROOT/deploy/docker/Dockerfile.cve-sync" "$ROOT"
  kind load docker-image --name "$CLUSTER" lyrahcle/backend:dev lyrahcle/frontend:dev lyrahcle/cve-sync:dev
fi

helm upgrade --install lyrahcle "$ROOT/deploy/helm/lyrahcle" \
  --kube-context "$CONTEXT" \
  --namespace "$NAMESPACE" \
  -f "$ROOT/deploy/helm/lyrahcle/values-staging.yaml" \
  --wait \
  --timeout 5m

kubectl --context "$CONTEXT" -n "$NAMESPACE" rollout status deployment/lyrahcle-backend --timeout=180s
kubectl --context "$CONTEXT" -n "$NAMESPACE" rollout status deployment/lyrahcle-frontend --timeout=180s
kubectl --context "$CONTEXT" -n "$NAMESPACE" rollout status deployment/lyrahcle-cve-sync --timeout=180s

printf '\nLyrahcle local Kubernetes environment is ready.\n'
printf 'Cluster: %s  Namespace: %s\n' "$CLUSTER" "$NAMESPACE"
printf 'UI port-forward: kubectl --context %s -n %s port-forward svc/lyrahcle-frontend 18080:80\n' "$CONTEXT" "$NAMESPACE"

if [[ "$RUN_SMOKE" == "1" ]]; then
  LYRAHCLE_KIND_CLUSTER="$CLUSTER" LYRAHCLE_K8S_NAMESPACE="$NAMESPACE" "$ROOT/scripts/k8s-smoke-test.sh"
fi
