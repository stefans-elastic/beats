The System `cpu` metricset provides CPU statistics.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* OpenBSD
* Windows


## Configuration [_configuration_5]

**`cpu.metrics`**
:   This option controls what CPU metrics are reported. The value is a list and three metric types are supported - `percentages`, `normalized_percentages`, and `ticks`. The default value is `cpu.metrics: [percentages]`.

**`use_performance_counters`**
:   This option enables the use of performance counters to collect data for the CPU/core metricset. It is only effective on Windows. You should use this option if running beats on machins with more than 64 cores. The default value is `use_performance_counters: true`

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: [cpu]
      cpu.metrics: [percentages, normalized_percentages, ticks]
      #use_performance_counters: false
    ```
