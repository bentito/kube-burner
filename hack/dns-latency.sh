#!/usr/bin/env bash
set -euo pipefail

KUBEBURNER=${KUBEBURNER:-kube-burner}
JOB_FILE=$(dirname "$0")/../examples/workloads/dns-latency/dns-latency.yml

$KUBEBURNER init -c "$JOB_FILE" "$@"
