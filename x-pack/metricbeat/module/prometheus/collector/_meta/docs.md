The Prometheus `collector` metricset scrapes data from [prometheus exporters](https://prometheus.io/docs/instrumenting/exporters/).


## Scraping from a Prometheus exporter [_scraping_from_a_prometheus_exporter]

To scrape metrics from a Prometheus exporter, configure the `hosts` field to it. The path to retrieve the metrics from (`/metrics` by default) can be configured with `metrics_path`.

```yaml
- module: prometheus
  period: 10s
  hosts: ["node:9100"]
  metrics_path: /metrics

  #username: "user"
  #password: "secret"

  # This can be used for service account based authorization:
  #bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  #ssl.certificate_authorities:
  #  - /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
```


## Histograms and types [_histograms_and_types]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


```yaml
metricbeat.modules:
- module: prometheus
  period: 10s
  hosts: ["localhost:9090"]
  use_types: true
  rate_counters: false
```

`use_types` parameter (default: false) enables a different layout for metrics storage, leveraging Elasticsearch types, including [histograms](elasticsearch://reference/elasticsearch/mapping-reference/histogram.md).

`rate_counters` parameter (default: false) enables calculating a rate out of Prometheus counters. When enabled, Metricbeat stores the counter increment since the last collection. This metric should make some aggregations easier and with better performance. This parameter can only be enabled in combination with `use_types`.

When `use_types` and `rate_counters` are enabled, metrics are stored like this:

```json
{
    "prometheus": {
        "labels": {
            "instance": "172.27.0.2:9090",
            "job": "prometheus"
        },
        "prometheus_target_interval_length_seconds_count": {
            "counter": 1,
            "rate": 0
        },
        "prometheus_target_interval_length_seconds_sum": {
            "counter": 15.000401344,
            "rate": 0
        }
        "prometheus_tsdb_compaction_chunk_range_seconds_bucket": {
            "histogram": {
                "values": [50, 300, 1000, 4000, 16000],
                "counts": [10, 2, 34, 7]
            }
        }
    },
}
```


## Scraping all metrics from a Prometheus server [_scraping_all_metrics_from_a_prometheus_server]

::::{warning}
Depending on your scale this method may not be suitable. We recommend using the [remote_write](/reference/metricbeat/metricbeat-metricset-prometheus-remote_write.md) metricset for this, and make Prometheus push metrics to Metricbeat.

::::


This module can scrape all metrics stored in a Prometheus server, by using the [federation API](https://prometheus.io/docs/prometheus/latest/federation/). By pointing this config to the Prometheus server:

```yaml
metricbeat.modules:
- module: prometheus
  period: 10s
  hosts: ["localhost:9090"]
  metrics_path: '/federate'
  query:
    'match[]': '{__name__!=""}'
```

::::{note}
federation API returns all metrics as untyped, as a result even in case `use_types` and `rate_counters` parameters are enabled, rate metrics will NOT be calculated out of Prometheus counters. To get rate metrics calculated should be used [remote_write](/reference/metricbeat/metricbeat-metricset-prometheus-remote_write.md) metricset instead.
::::



## Filtering metrics [_filtering_metrics_2]

In order to filter out/in metrics one can make use of `metrics_filters.include` `metrics_filters.exclude` settings:

```yaml
- module: prometheus
  period: 10s
  hosts: ["localhost:9090"]
  metrics_path: /metrics
  metrics_filters:
    include: ["node_filesystem_*"]
    exclude: ["node_filesystem_device_*"]
```

The configuration above will include only metrics that match `node_filesystem_*` pattern and do not match `node_filesystem_device_*`.

To keep only specific metrics, anchor the start and the end of the regexp of each metric:

* the caret `^` matches the beginning of a text or line,
* the dollar sign `$` matches the end of a text.

```yaml
- module: prometheus
  period: 10s
  hosts: ["localhost:9090"]
  metrics_path: /metrics
  metrics_filters:
    include: ["^node_network_net_dev_group$", "^node_network_up$"]
```
