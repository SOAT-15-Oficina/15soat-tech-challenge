#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBE_NAMESPACE="${KUBE_NAMESPACE:-workshop}"
HPA_NAME="${HPA_NAME:-api-hpa}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-api}"
SERVICE_NAME="${SERVICE_NAME:-api-service}"
SERVICE_PORT="${SERVICE_PORT:-8080}"
LOAD_PROXY_IMAGE="${LOAD_PROXY_IMAGE:-alpine/socat:1.8.0.3}"
LOAD_PROXY_POD="hpa-load-proxy-$$"
MIN_REPLICAS="${MIN_REPLICAS:-2}"
LOCAL_PORT="${LOCAL_PORT:-18080}"
METRICS_TIMEOUT="${METRICS_TIMEOUT:-180}"
BASELINE_TIMEOUT="${BASELINE_TIMEOUT:-600}"
SCALE_OUT_TIMEOUT="${SCALE_OUT_TIMEOUT:-300}"
SCALE_DOWN_TIMEOUT="${SCALE_DOWN_TIMEOUT:-600}"
OBSERVE_SCALE_DOWN="${OBSERVE_SCALE_DOWN:-true}"
if [[ -n "${BASE_URL:-}" ]]; then
  BASE_URL="${BASE_URL%/}"
  USE_CLUSTER_PROXY="${USE_CLUSTER_PROXY:-false}"
else
  BASE_URL="http://127.0.0.1:${LOCAL_PORT}"
  USE_CLUSTER_PROXY="${USE_CLUSTER_PROXY:-true}"
fi
WORK_ORDER_CODE="${WORK_ORDER_CODE:-OS-2026-0001}"
CUSTOMER_DOCUMENT="${CUSTOMER_DOCUMENT:-12345678901}"
K6_SCRIPT="${K6_SCRIPT:-${ROOT_DIR}/tests/load/hpa.js}"

PORT_FORWARD_PID=""
K6_PID=""
LOAD_PROXY_CREATED="false"

log() {
  printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

cleanup() {
  local pid
  for pid in "$K6_PID" "$PORT_FORWARD_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  if [[ "$LOAD_PROXY_CREATED" == "true" ]]; then
    kubectl delete pod "$LOAD_PROXY_POD" -n "$KUBE_NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

hpa_metrics_are_numeric() {
  local metrics scaling_active
  metrics="$(kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE" \
    -o jsonpath='{range .status.currentMetrics[*]}{.resource.current.averageUtilization}{"\n"}{end}' 2>/dev/null || true)"
  scaling_active="$(kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE" \
    -o jsonpath='{.status.conditions[?(@.type=="ScalingActive")].status}' 2>/dev/null || true)"

  [[ "$scaling_active" == "True" ]] || return 1
  [[ "$(printf '%s\n' "$metrics" | sed '/^$/d' | wc -l | tr -d ' ')" -eq 2 ]] || return 1
  ! printf '%s\n' "$metrics" | sed '/^$/d' | grep -Eqv '^[0-9]+$'
}

wait_for_metrics() {
  local deadline=$((SECONDS + METRICS_TIMEOUT))

  log "Waiting for metrics.k8s.io and numeric HPA metrics"
  until (( SECONDS >= deadline )); do
    if kubectl get --raw '/apis/metrics.k8s.io/v1beta1/nodes' >/dev/null 2>&1 \
      && kubectl top pods -n "$KUBE_NAMESPACE" >/dev/null 2>&1 \
      && hpa_metrics_are_numeric; then
      kubectl top pods -n "$KUBE_NAMESPACE"
      kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE"
      return 0
    fi
    sleep 5
  done

  kubectl get apiservice v1beta1.metrics.k8s.io -o wide || true
  kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE" || true
  printf 'HPA metrics remained unavailable or unknown for %ss.\n' "$METRICS_TIMEOUT" >&2
  return 1
}

wait_for_endpoint() {
  local deadline=$((SECONDS + 60))
  local url="${BASE_URL}/public/work-orders/${WORK_ORDER_CODE}?document=${CUSTOMER_DOCUMENT}"

  until (( SECONDS >= deadline )); do
    if curl --fail --silent --show-error "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done

  printf 'API did not become available at %s.\n' "$BASE_URL" >&2
  return 1
}

wait_for_baseline() {
  local deadline=$((SECONDS + BASELINE_TIMEOUT))
  local desired available

  log "Waiting for the baseline of ${MIN_REPLICAS} replicas before applying load"
  until (( SECONDS >= deadline )); do
    desired="$(kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE" \
      -o jsonpath='{.status.desiredReplicas}' 2>/dev/null || true)"
    available="$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$KUBE_NAMESPACE" \
      -o jsonpath='{.status.availableReplicas}' 2>/dev/null || true)"
    if [[ "$desired" == "$MIN_REPLICAS" && "$available" == "$MIN_REPLICAS" ]]; then
      log "Baseline confirmed with ${MIN_REPLICAS} available replicas"
      return 0
    fi
    log "Baseline pending: desired=${desired:-unknown}, available=${available:-unknown}"
    sleep 15
  done

  printf 'API did not return to the %s-replica baseline within %ss.\n' "$MIN_REPLICAS" "$BASELINE_TIMEOUT" >&2
  return 1
}

monitor_scale_out() {
  local deadline=$((SECONDS + SCALE_OUT_TIMEOUT))
  local replicas metrics unknown_samples=0

  log "Monitoring HPA for scale-out above ${MIN_REPLICAS} available replicas"
  until (( SECONDS >= deadline )); do
    replicas="$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$KUBE_NAMESPACE" \
      -o jsonpath='{.status.availableReplicas}' 2>/dev/null || true)"
    replicas="${replicas:-0}"
    metrics="$(kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE" \
      -o custom-columns='CPU:.status.currentMetrics[0].resource.current.averageUtilization,MEMORY:.status.currentMetrics[1].resource.current.averageUtilization,DESIRED:.status.desiredReplicas,CURRENT:.status.currentReplicas' \
      --no-headers 2>/dev/null || true)"
    log "HPA ${metrics:-unavailable}; available replicas=${replicas}"

    if hpa_metrics_are_numeric; then
      unknown_samples=0
    else
      unknown_samples=$((unknown_samples + 1))
      if (( unknown_samples >= 3 )); then
        printf 'HPA metrics became unknown during the load test.\n' >&2
        return 2
      fi
    fi

    if [[ "$replicas" =~ ^[0-9]+$ ]] && (( replicas > MIN_REPLICAS )); then
      log "Scale-out confirmed with ${replicas} available replicas"
      return 0
    fi

    if ! kill -0 "$K6_PID" 2>/dev/null; then
      printf 'k6 finished before the API exceeded %s available replicas.\n' "$MIN_REPLICAS" >&2
      return 1
    fi
    sleep 5
  done

  printf 'API did not exceed %s available replicas within %ss.\n' "$MIN_REPLICAS" "$SCALE_OUT_TIMEOUT" >&2
  return 1
}

observe_scale_down() {
  local deadline=$((SECONDS + SCALE_DOWN_TIMEOUT))
  local desired

  [[ "$OBSERVE_SCALE_DOWN" == "true" ]] || return 0
  log "Observing scale-down to ${MIN_REPLICAS} replicas (this can take several minutes)"
  until (( SECONDS >= deadline )); do
    desired="$(kubectl get hpa "$HPA_NAME" -n "$KUBE_NAMESPACE" \
      -o jsonpath='{.status.desiredReplicas}' 2>/dev/null || true)"
    log "HPA desired replicas=${desired:-unknown}"
    if [[ "$desired" == "$MIN_REPLICAS" ]]; then
      kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$KUBE_NAMESPACE" --timeout=120s
      log "Scale-down to the minimum replica count confirmed"
      return 0
    fi
    sleep 15
  done

  log "Scale-down was not observed within ${SCALE_DOWN_TIMEOUT}s; inspect with kubectl get hpa ${HPA_NAME} -n ${KUBE_NAMESPACE} -w"
}

for command in kubectl k6 curl; do
  require_command "$command"
done

kubectl cluster-info >/dev/null
kubectl get deployment "$DEPLOYMENT_NAME" -n "$KUBE_NAMESPACE" >/dev/null
kubectl get pods -n "$KUBE_NAMESPACE"
kubectl rollout status deployment/metrics-server -n kube-system --timeout="${METRICS_TIMEOUT}s"
kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$KUBE_NAMESPACE" --timeout=180s
wait_for_metrics
wait_for_baseline

if [[ "$USE_CLUSTER_PROXY" == "true" ]]; then
  log "Starting temporary in-cluster TCP proxy for ${SERVICE_NAME}"
  kubectl run "$LOAD_PROXY_POD" -n "$KUBE_NAMESPACE" \
    --image="$LOAD_PROXY_IMAGE" \
    --restart=Never \
    --labels=app.kubernetes.io/name=hpa-load-proxy \
    --command -- socat \
    "TCP-LISTEN:8080,fork,reuseaddr" \
    "TCP:${SERVICE_NAME}.${KUBE_NAMESPACE}.svc.cluster.local:${SERVICE_PORT}"
  LOAD_PROXY_CREATED="true"
  kubectl wait --for=condition=Ready "pod/${LOAD_PROXY_POD}" -n "$KUBE_NAMESPACE" --timeout=120s

  log "Starting port-forward through the cluster proxy on ${BASE_URL}"
  kubectl port-forward "pod/${LOAD_PROXY_POD}" "${LOCAL_PORT}:8080" -n "$KUBE_NAMESPACE" >/dev/null 2>&1 &
  PORT_FORWARD_PID=$!
else
  log "Using externally supplied BASE_URL=${BASE_URL}"
fi
wait_for_endpoint

log "Starting k6 load test for ${WORK_ORDER_CODE}"
BASE_URL="$BASE_URL" WORK_ORDER_CODE="$WORK_ORDER_CODE" CUSTOMER_DOCUMENT="$CUSTOMER_DOCUMENT" \
  k6 run --quiet "$K6_SCRIPT" &
K6_PID=$!

set +e
monitor_scale_out
SCALE_STATUS=$?
wait "$K6_PID"
K6_STATUS=$?
set -e
K6_PID=""

if (( K6_STATUS != 0 )); then
  printf 'k6 failed or one of its thresholds was violated.\n' >&2
  exit "$K6_STATUS"
fi
if (( SCALE_STATUS != 0 )); then
  exit "$SCALE_STATUS"
fi

wait_for_metrics
observe_scale_down
log "HPA validation completed successfully"
