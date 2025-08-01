---
navigation_title: "Understand logged metrics"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/understand-auditbeat-logs.html
applies_to:
  stack: ga
---

# Understand metrics in Auditbeat logs [understand-auditbeat-logs]


Every 30 seconds (by default), Auditbeat collects a *snapshot* of metrics about itself. From this snapshot, Auditbeat computes a *delta snapshot*; this delta snapshot contains any metrics that have *changed* since the last snapshot. Note that the values of the metrics are the values when the snapshot is taken, *NOT* the *difference* in values from the last snapshot.

If this delta snapshot contains *any* metrics (indicating at least one metric that has changed since the last snapshot), this delta snapshot is serialized as JSON and emitted in Auditbeat’s logs at the `INFO` log level. Most snapshot fields report the change in the metric since the last snapshot, however some fields are *gauges*, which always report the current value. Here is an example of such a log entry:

```json
{"log.level":"info","@timestamp":"2023-07-14T12:50:36.811Z","log.logger":"monitoring","log.origin":{"file.name":"log/log.go","file.line":187},"message":"Non-zero metrics in the last 30s","service.name":"filebeat","monitoring":{"metrics":{"beat":{"cgroup":{"memory":{"mem":{"usage":{"bytes":0}}}},"cpu":{"system":{"ticks":692690,"time":{"ms":60}},"total":{"ticks":3167250,"time":{"ms":150},"value":3167250},"user":{"ticks":2474560,"time":{"ms":90}}},"handles":{"limit":{"hard":1048576,"soft":1048576},"open":32},"info":{"ephemeral_id":"2bab8688-34c0-4522-80af-db86948d547d","uptime":{"ms":617670096},"version":"8.6.2"},"memstats":{"gc_next":57189272,"memory_alloc":43589824,"memory_total":275281335792,"rss":183574528},"runtime":{"goroutines":212}},"filebeat":{"events":{"active":5,"added":52,"done":49},"harvester":{"open_files":6,"running":6,"started":1}},"libbeat":{"config":{"module":{"running":15}},"output":{"events":{"acked":48,"active":0,"batches":6,"total":48},"read":{"bytes":210},"write":{"bytes":26923}},"pipeline":{"clients":15,"events":{"active":5,"filtered":1,"published":51,"total":52},"queue":{"max_events":3500,"filled":{"events":5,"bytes":6425,"pct":0.0014},"added":{"events":52,"bytes":65702},"consumed":{"events":52,"bytes":65702},"removed":{"events":48,"bytes":59277},"acked":48}}},"registrar":{"states":{"current":14,"update":49},"writes":{"success":6,"total":6}},"system":{"load":{"1":0.91,"15":0.37,"5":0.4,"norm":{"1":0.1138,"15":0.0463,"5":0.05}}}},"ecs.version":"1.6.0"}}
```


## Details [_details]

Focussing on the `.monitoring.metrics` field, and formatting the JSON, it’s value is:

```json
{
  "beat": {
    "cgroup": {
      "memory": {
        "mem": {
          "usage": {
            "bytes": 0
          }
        }
      }
    },
    "cpu": {
      "system": {
        "ticks": 692690,
        "time": {
          "ms": 60
        }
      },
      "total": {
        "ticks": 3167250,
        "time": {
          "ms": 150
        },
        "value": 3167250
      },
      "user": {
        "ticks": 2474560,
        "time": {
          "ms": 90
        }
      }
    },
    "handles": {
      "limit": {
        "hard": 1048576,
        "soft": 1048576
      },
      "open": 32
    },
    "info": {
      "ephemeral_id": "2bab8688-34c0-4522-80af-db86948d547d",
      "uptime": {
        "ms": 617670096
      },
      "version": "8.6.2"
    },
    "memstats": {
      "gc_next": 57189272,
      "memory_alloc": 43589824,
      "memory_total": 275281335792,
      "rss": 183574528
    },
    "runtime": {
      "goroutines": 212
    }
  },
  "filebeat": {
    "events": {
      "active": 5,
      "added": 52,
      "done": 49
    },
    "harvester": {
      "open_files": 6,
      "running": 6,
      "started": 1
    }
  },
  "libbeat": {
    "config": {
      "module": {
        "running": 15
      }
    },
    "output": {
      "events": {
        "acked": 48,
        "active": 0,
        "batches": 6,
        "total": 48
      },
      "read": {
        "bytes": 210
      },
      "write": {
        "bytes": 26923
      }
    },
    "pipeline": {
      "clients": 15,
      "events": {
        "active": 5,
        "filtered": 1,
        "published": 51,
        "total": 52
      },
      "queue": {
        "max_events": 3500,
        "filled": {
          "events": 5,
          "bytes": 6425,
          "pct": 0.0014
        },
        "added": {
          "events": 52,
          "bytes": 65702
        },
        "consumed": {
          "events": 52,
          "bytes": 65702
        },
        "removed": {
          "events": 48,
          "bytes": 59277
        },
        "acked": 48
      }
    }
  },
  "registrar": {
    "states": {
      "current": 14,
      "update": 49
    },
    "writes": {
      "success": 6,
      "total": 6
    }
  },
  "system": {
    "load": {
      "1": 0.91,
      "15": 0.37,
      "5": 0.4,
      "norm": {
        "1": 0.1138,
        "15": 0.0463,
        "5": 0.05
      }
    }
  }
}
```

The following tables explain the meaning of the most important fields under `.monitoring.metrics` and also provide hints that might be helpful in troubleshooting Auditbeat issues.

| Field path (relative to `.monitoring.metrics`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.beat` | Object | Information that is common to all Beats, e.g. version, goroutines, file handles, CPU, memory |  |
| `.libbeat` | Object | Information about the publisher pipeline and output, also common to all Beats |  |

| Field path (relative to `.monitoring.metrics.beat`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.runtime.goroutines` | Integer | Number of goroutines running | If this number grows over time, it indicates a goroutine leak |

| Field path (relative to `.monitoring.metrics.libbeat`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.pipeline.events.active` | Integer | Number of events currently in the libbeat publisher pipeline. | If this number grows over time, it may indicate that Auditbeat is producing events faster than the output can consume them. Consider increasing the number of output workers (if this setting is supported by the output; {{es}} and {{ls}} outputs support this setting). The pipeline includes events currently being processed as well as events in the queue. So this metric can sometimes end up slightly higher than the queue size. If this metric reaches the maximum queue size (`queue.mem.events` for the in-memory queue), it almost certainly indicates backpressure on Auditbeat, implying that Auditbeat may need to temporarily stop ingesting more events from the source until this backpressure is relieved. |
| `.output.events.total` | Integer | Number of events currently being processed by the output. | If this number grows over time, it may indicate that the output destination (e.g. {{ls}} pipeline or {{es}} cluster) is not able to accept events at the same or faster rate than what Auditbeat is sending to it. |
| `.output.events.acked` | Integer | Number of events acknowledged by the output destination. | Generally, we want this number to be the same as `.output.events.total` as this indicates that the output destination has reliably received all the events sent to it. |
| `.output.events.failed` | Integer | Number of events that Auditbeat tried to send to the output destination, but the destination failed to receive them. | Generally, we want this field to be absent or its value to be zero. When the value is greater than zero, it’s useful to check Auditbeat’s logs right before this log entry’s `@timestamp` to see if there are any connectivity issues with the output destination. Note that failed events are not lost or dropped; they will be sent back to the publisher pipeline for retrying later. |
| `.output.events.dropped` | Integer | Number of events that Auditbeat gave up sending to the output destination because of a permanent (non-retryable) error. |
| `.output.events.dead_letter` | Integer | Number of events that Auditbeat successfully sent to a configured dead letter index after they failed to ingest in the primary index. |
| `.output.write.latency` | Object | Reports statistics on the time to send an event to the connected output, in milliseconds. This can be used to diagnose delays and performance issues caused by I/O or output configuration. This metric is available for the Elasticsearch, file, redis, and logstash outputs. |

| Field path (relative to `.monitoring.metrics.libbeat.pipeline`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.queue.max_events` | Integer (gauge) | The queue's maximum event count if it has one, otherwise zero. |
| `.queue.max_bytes` | Integer (gauge) | The queue's maximum byte count if it has one, otherwise zero. |
| `.queue.filled.events` | Integer (gauge) | Number of events currently stored by the queue. |
| `.queue.filled.bytes` | Integer (gauge) | Number of bytes currently stored by the queue. |
| `.queue.filled.pct` | Float (gauge) | How full the queue is relative to its maximum size, as a fraction from 0 to 1. | Low throughput while `queue.filled.pct` is low means congestion in the input. Low throughput while `queue.filled.pct` is high means congestion in the output.
| `.queue.added.events` | Integer | Number of events added to the queue by input workers. |
| `.queue.added.bytes` | Integer | Number of bytes added to the queue by input workers. |
| `.queue.consumed.events` | Integer | Number of events sent to output workers. |
| `.queue.consumed.bytes` | Integer | Number of bytes sent to output workers. |
| `.queue.removed.events` | Integer | Number of events removed from the queue after being processed by output workers. |
| `.queue.removed.bytes` | Integer | Number of bytes removed from the queue after being processed by output workers. |

When using the memory queue, byte metrics are only set if the output supports them. Currently only the Elasticsearch output supports byte metrics.


## Useful commands [_useful_commands_2]


### Parse monitoring metrics from unstructured Auditbeat logs [_parse_monitoring_metrics_from_unstructured_auditbeat_logs]

For Auditbeat versions that emit unstructured logs, the following script can be used to parse monitoring metrics from such logs: [https://github.com/elastic/beats/blob/main/script/metrics_from_log_file.sh](https://github.com/elastic/beats/blob/main/script/metrics_from_log_file.sh).

