# opentelemetry-stack-charts

## Requirements

Install certmanager (required by Otel operator):

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

Install the OpenTelemetry Operator helm charts (would not work as a dependency since required CRs are deployed on the fly in an admission webhook, causing our local templates to fail)

```bash
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
help repo update 
helm install -n opentelemetry-operator-system --create-namespace opentelemetry-operator open-telemetry/opentelemetry-operator
```

### Monitoring

### Rancher monitoring setup

Reference : 
https://prometheus.io/docs/guides/opentelemetry/

Required version is whatever includes https://github.com/rancher/ob-team-charts/pull/110

Required values:
```yaml
prometheus:
    disableServiceDiscovery : true
    enableOTLPReceiver : true
    tsdb:
        outOfOrderTimeWindow: 300s
    # ...
```

Playing around with the following configuration may also prove useful:
```yaml
prometheus:
    otlp:
        promoteResourceAttributes:
        - service.instance.id
        - service.name
        - service.namespace
        - cloud.availability_zone
        - cloud.region
        - container.name
        - deployment.environment.name
        - k8s.cluster.name
        - k8s.container.name
        - k8s.cronjob.name
        - k8s.daemonset.name
        - k8s.deployment.name
        - k8s.job.name
        - k8s.namespace.name
        - k8s.pod.name
        - k8s.replicaset.name
        - k8s.statefulset.name
        translationStrategy: NoUTF8EscapingWithSuffixes
```

```
helm install otel-stack -n otel-stack --create-namespace ./ --set metricsCollector.enabled=true
```


## Grafana dashboards

For debugging opentelemetry collector using the self-telemetry metrics, use:
https://grafana.com/grafana/dashboards/15983-opentelemetry-collector/


## Analyzing metric differences


Build the hacky tool for analyzing differences:

```bash
cd hack && make build
```


```bash
kubectl port-forward -n cattle-monitoring-system svc/rancher-monitoring-prometheus 9090:9090
```

Setup monitoring with default prometheus collection:

```bash
cd hack && ./bin/tools -d prometheus 

```
Setup monitoring with otel metrics collection (wait a couple of minutes to wait for collectors to start and ingest all relevant metrics):

```bash
cd hack && ./bin/tools -d otel
```

Then run the command
```bash
cd hack && ./bin/tools compare
```


As of writing this, the meaningful differences are that otel collector adds `total` suffixes by default to counters:
```json
[
    "apiextensions_openapi_v2_regeneration_count_total",
    "apiextensions_openapi_v3_regeneration_count_total",
    "apiserver_admission_webhook_fail_open_count_total",
    "apiserver_admission_webhook_rejection_count_total",
    "apiserver_egress_dialer_dial_failure_count_total",
    "authenticated_user_requests_total",
    "authentication_attempts_total",
    "container_memory_failcnt_total",
    "endpoint_slice_controller_changes_total",
    "endpoint_slice_controller_syncs_total",
    "endpoint_slice_mirroring_controller_changes_total",
    "go_cpu_classes_cpu_seconds_total",
    "go_cpu_classes_gc_cpu_seconds_total",
    "go_cpu_classes_scavenge_cpu_seconds_total",
    "go_gc_cycles_gc_cycles_total",
    "go_sync_mutex_wait_seconds_total",
    "kube_poddisruptionbudget_created",
    "kube_poddisruptionbudget_status_current_healthy",
    "kube_poddisruptionbudget_status_desired_healthy",
    "kube_poddisruptionbudget_status_expected_pods",
    "kube_poddisruptionbudget_status_observed_generation",
    "kube_poddisruptionbudget_status_pod_disruptions_allowed",
    "kubelet_evented_pleg_connection_error_count_total",
    "kubelet_evented_pleg_connection_success_count_total",
    "kubelet_pleg_discard_events_total",
    "lasso_controller_handler_execution_total"
]
```

where as prometheus does not by default:
```json
[
    "apiextensions_openapi_v2_regeneration_count",
    "apiextensions_openapi_v3_regeneration_count",
    "apiserver_admission_webhook_fail_open_count",
    "apiserver_admission_webhook_rejection_count",
    "apiserver_egress_dialer_dial_failure_count",
    "authenticated_user_requests",
    "authentication_attempts",
    "container_memory_failcnt",
    "endpoint_slice_controller_changes",
    "endpoint_slice_controller_syncs",
    "endpoint_slice_mirroring_controller_changes",
    "go_cpu_classes_gc_total_cpu_seconds_total",
    "go_cpu_classes_scavenge_total_cpu_seconds_total",
    "go_cpu_classes_total_cpu_seconds_total",
    "go_gc_cycles_total_gc_cycles_total",
    "go_sync_mutex_wait_total_seconds_total",
    "kubelet_evented_pleg_connection_error_count",
    "kubelet_evented_pleg_connection_success_count",
    "kubelet_pleg_discard_events",
    "lasso_controller_total_handler_execution",
]
```

Also note that the collector instances are not able to scrape with TLS configs at the moment, so for example the otel collector will not be able to collect:


A fork of the upstream operator will be required to support that functionality.
```json
[
    "prometheus_operator_build_info",
    "prometheus_operator_feature_gate",
    "prometheus_operator_kubelet_managed_resource",
    "prometheus_operator_kubernetes_client_http_request_duration_seconds_count",
    "prometheus_operator_kubernetes_client_http_request_duration_seconds_sum",
    "prometheus_operator_kubernetes_client_http_requests_total",
    "prometheus_operator_kubernetes_client_rate_limiter_duration_seconds_count",
    "prometheus_operator_kubernetes_client_rate_limiter_duration_seconds_sum",
    "prometheus_operator_list_operations_failed_total",
    "prometheus_operator_list_operations_total",
    "prometheus_operator_managed_resources",
    "prometheus_operator_node_address_lookup_errors_total",
    "prometheus_operator_node_syncs_failed_total",
    "prometheus_operator_node_syncs_total",
    "prometheus_operator_ready",
    "prometheus_operator_reconcile_duration_seconds_bucket",
    "prometheus_operator_reconcile_duration_seconds_count",
    "prometheus_operator_reconcile_duration_seconds_sum",
    "prometheus_operator_reconcile_errors_total",
    "prometheus_operator_reconcile_operations_total",
    "prometheus_operator_reconcile_sts_delete_create_total",
    "prometheus_operator_spec_replicas",
    "prometheus_operator_spec_shards",
    "prometheus_operator_status_update_errors_total",
    "prometheus_operator_status_update_operations_total",
    "prometheus_operator_syncs",
    "prometheus_operator_triggered_total",
    "prometheus_operator_watch_operations_failed_total",
    "prometheus_operator_watch_operations_total",
    "prometheus_target_metadata_cache_bytes",
    "prometheus_target_metadata_cache_entries",
    "prometheus_target_scrape_pool_symboltable_items",
    "prometheus_target_scrape_pool_sync_total",
    "prometheus_target_scrape_pool_target_limit",
    "prometheus_target_scrape_pool_targets",
    "prometheus_target_sync_failed_total",
    "prometheus_target_sync_length_seconds",
    "prometheus_target_sync_length_seconds_count",
    "prometheus_target_sync_length_seconds_sum"
]
```