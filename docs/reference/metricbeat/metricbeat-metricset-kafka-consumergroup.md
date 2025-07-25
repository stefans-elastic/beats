---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-kafka-consumergroup.html
---

% This file is generated! See scripts/docs_collector.py

# Kafka consumergroup metricset [metricbeat-metricset-kafka-consumergroup]

This is the `consumergroup` metricset of the Kafka module.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-kafka.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "kafka.consumergroup",
        "duration": 115000,
        "module": "kafka"
    },
    "kafka": {
        "broker": {
            "address": "172.21.0.2:9092",
            "id": 0
        },
        "consumergroup": {
            "client": {
                "host": "127.0.0.1",
                "id": "consumer-1",
                "member_id": "consumer-1-8653cb3a-afed-4b1b-87d0-2a208319b41e"
            },
            "consumer_lag": 77,
            "error": {
                "code": 0
            },
            "id": "console-consumer-40539",
            "meta": "",
            "offset": -1
        },
        "partition": {
            "id": 0,
            "topic_id": "0-test"
        },
        "topic": {
            "name": "test"
        }
    },
    "metricset": {
        "name": "consumergroup",
        "period": 10000
    },
    "service": {
        "address": "172.21.0.2:9092",
        "type": "kafka"
    }
}
```
