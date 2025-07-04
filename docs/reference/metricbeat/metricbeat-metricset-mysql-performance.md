---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-mysql-performance.html
---

% This file is generated! See scripts/docs_collector.py

# MySQL performance metricset [metricbeat-metricset-mysql-performance]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


`performance` metricset fetches performance related metrics (events statements and table io waits) from MySQL

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-mysql.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2024-02-27T07:33:02.881Z",
    "@metadata": {
        "beat": "metricbeat",
        "type": "_doc",
        "version": "8.14.0"
    },
    "mysql": {
        "performance": {
            "events_statements": {
                "digest": {
                    "text": "SHOW SCHEMAS"
                },
                "count": {
                    "star": 5
                },
                "avg": {
                    "timer": {
                        "wait": 1.6439131e+10
                    }
                },
                "max": {
                    "timer": {
                        "wait": 4.0834164e+10
                    }
                },
                "last": {
                    "seen": "2024-02-27 06:44:17.296246"
                },
                "quantile": {
                    "95": 4.1686938347e+10
                },
                "schemaname": "performance_schema"
            }
        }
    },
    "host": {
        "id": "41359f29035549cda159ae8d1a533d72",
        "containerized": false,
        "ip": [
            "127.0.0.1"
        ],
        "name": "localhost",
        "mac": [
            "86-32-76-45-EB-2B"
        ],
        "hostname": "localhost",
        "architecture": "x86_64",
        "os": {
            "name": "CentOS Linux",
            "kernel": "3.10.0-1160.102.1.el7.x86_64",
            "codename": "Core",
            "type": "linux",
            "platform": "centos",
            "version": "7 (Core)",
            "family": "redhat"
        }
    },
    "agent": {
        "type": "metricbeat",
        "version": "8.14.0",
        "ephemeral_id": "539a163b-91ab-433c-9893-31a48d09b5a7",
        "id": "e5bcfbf0-4c74-44dd-b711-c5e90a69ab7a",
        "name": "localhost"
    },
    "ecs": {
        "version": "8.0.0"
    },
    "event": {
        "dataset": "mysql.performance",
        "module": "mysql",
        "duration": 14244062
    },
    "metricset": {
        "period": 10000,
        "name": "performance"
    },
    "service": {
        "address": "tcp(127.0.0.1:3306)/?readTimeout=10s&timeout=10s&writeTimeout=10s",
        "type": "mysql"
    }
}
```
