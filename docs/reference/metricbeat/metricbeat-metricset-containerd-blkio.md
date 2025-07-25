---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-containerd-blkio.html
---

% This file is generated! See scripts/docs_collector.py

# Containerd blkio metricset [metricbeat-metricset-containerd-blkio]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the blkio metricset of the module containerd.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-containerd.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2019-03-01T08:05:34.853Z",
    "container": {
        "id": "7434687dbe3684407afa899582f2909203b9dc5537632b512f76798db5c0787d"
    },
    "containerd": {
        "blkio": {
            "device": "/dev/vda",
            "read": {
                "bytes": 69246976,
                "ops": 830
            },
            "summary": {
                "bytes": 69271552,
                "ops": 832
            },
            "write": {
                "bytes": 24576,
                "ops": 2
            }
        },
        "namespace": "k8s.io"
    },
    "event": {
        "dataset": "containerd.blkio",
        "duration": 115000,
        "module": "containerd"
    },
    "metricset": {
        "name": "blkio",
        "period": 10000
    },
    "service": {
        "address": "127.0.0.1:55555",
        "type": "containerd"
    }
}
```
