# DNS latency troubleshooting with kube-burner

This guide walks through running the `dns-latency` workload to measure DNS resolution latency on a Kubernetes cluster.

## Prerequisites

- Access to a Kubernetes cluster. The job works on clusters such as kind or OpenShift.
- `kubectl` configured to talk to the cluster.
- The `kube-burner` binary available in your `$PATH`.

## Running on kind

```bash
kind create cluster
```

The workload can also target any existing cluster by ensuring `KUBECONFIG` points to it.

## Launch the workload

From the repository root run:

```bash
hack/dns-latency.sh
```

This script invokes kube-burner with the [dns-latency job](examples/workloads/dns-latency/dns-latency.yml) which
spawns a `dnsperf` pod that repeatedly queries the cluster DNS service.

## Results

Metrics are written to the local `kube-burner/` directory. Files prefixed with `dnsLatencyMeasurement` contain
average, minimum and maximum query latency extracted from the `dnsperf` run.

These numbers can help identify slow DNS responses or network path issues.
