---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-coredns-stats.html
---

% This file is generated! See scripts/docs_collector.py

# Coredns stats metricset [metricbeat-metricset-coredns-stats]

This is the stats metricset of the module coredns.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-coredns.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2019-03-01T08:05:34.853Z",
    "coredns": {
        "stats": {
            "dns": {
                "request": {
                    "size": {
                        "bytes": {
                            "bucket": {
                                "+Inf": 440,
                                "0": 0,
                                "100": 440,
                                "1023": 440,
                                "16000": 440,
                                "200": 440,
                                "2047": 440,
                                "300": 440,
                                "32000": 440,
                                "400": 440,
                                "4095": 440,
                                "48000": 440,
                                "511": 440,
                                "64000": 440,
                                "8291": 440
                            },
                            "count": 440,
                            "sum": 22880
                        }
                    }
                },
                "response": {
                    "size": {
                        "bytes": {
                            "bucket": {
                                "+Inf": 440,
                                "0": 0,
                                "100": 440,
                                "1023": 440,
                                "16000": 440,
                                "200": 440,
                                "2047": 440,
                                "300": 440,
                                "32000": 440,
                                "400": 440,
                                "4095": 440,
                                "48000": 440,
                                "511": 440,
                                "64000": 440,
                                "8291": 440
                            },
                            "count": 440,
                            "sum": 29480
                        }
                    }
                }
            },
            "proto": "udp",
            "server": "dns://:53",
            "zone": "."
        }
    },
    "event": {
        "dataset": "coredns.stats",
        "duration": 115000,
        "module": "coredns"
    },
    "metricset": {
        "name": "stats",
        "period": 10000
    },
    "service": {
        "address": "127.0.0.1:55555",
        "type": "coredns"
    }
}
```
