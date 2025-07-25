:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/kafka/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `kafka` module collects and parses the logs created by [Kafka](https://kafka.apache.org/).

The module has additional support for parsing thread ID from logs.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_19]

The `kafka` module was tested with logs from versions 0.9, 1.1.0 and 2.0.0.


## Configure the module [configuring-kafka-module]

You can further refine the behavior of the `kafka` module by specifying [variable settings](#kafka-settings) in the `modules.d/kafka.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/kafka.yml` file to override the default paths for logs:

```yaml
- module: kafka
  log:
    enabled: true
    var.paths:
      - "/path/to/logs/controller.log*"
      - "/path/to/logs/server.log*"
      - "/path/to/logs/state-change.log*"
      - "/path/to/logs/kafka-*.log*"
```

To specify the same settings at the command line, you use:

```yaml
-M "kafka.log.var.paths=[/path/to/logs/controller.log*, /path/to/logs/server.log*, /path/to/logs/state-change.log*, /path/to/logs/kafka-*.log*]"
```


### Variable settings [kafka-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `kafka` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `kafka.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_5]

**`var.kafka_home`**
:   The path to your Kafka installation. The default is `/opt`. For example:

    ```yaml
    - module: kafka
      log:
        enabled: true
        var.kafka_home: /usr/share/kafka_2.12-2.4.0
        ...
    ```


**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_8]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Example dashboard [_example_dashboard_12]

This module comes with a sample dashboard to see Kafka logs and stack traces.

% TO DO: Use `:class: screenshot`
![filebeat kafka logs overview](images/filebeat-kafka-logs-overview.png)
