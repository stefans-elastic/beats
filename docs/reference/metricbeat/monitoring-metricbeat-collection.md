---
navigation_title: "Use {{metricbeat}} collection"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/monitoring-metricbeat-collection.html
applies_to:
  stack: ga
---

# Use {{metricbeat}} to send monitoring data [monitoring-metricbeat-collection]


In 7.3 and later, you can use {{metricbeat}} to collect data about Metricbeat and ship it to the monitoring cluster. The benefit of using {{metricbeat}} instead of internal collection is that the monitoring agent remains active even if the Metricbeat instance dies.

Because you’ll be using {{metricbeat}} to *monitor* Metricbeat, you’ll need to run two instances of Metricbeat: a main instance that collects metrics from the system and services running on the server, and a second instance that collects metrics from Metricbeat only. Using a separate instance as a monitoring agent allows you to send monitoring data to a dedicated monitoring cluster. If the main agent goes down, the monitoring agent remains active.

If you’re running Metricbeat as a service, this approach requires extra work because you need to run two instances of the same installed  service concurrently. If you don’t want to run two instances concurrently, use [internal collection](/reference/metricbeat/monitoring-internal-collection.md) instead of using {{metricbeat}}.

To collect and ship monitoring data:

1. [Configure the shipper you want to monitor](#configure-shipper)
2. [Install and configure {{metricbeat}} to collect monitoring data](#configure-metricbeat)


## Configure the shipper you want to monitor [configure-shipper]

1. Enable the HTTP endpoint to allow external collection of monitoring data:

    Add the following setting in the Metricbeat configuration file (`metricbeat.yml`):

    ```yaml
    http.enabled: true
    ```

    By default, metrics are exposed on port 5066. If you need to monitor multiple {{beats}} shippers running on the same server, set `http.port` to expose metrics for each shipper on a different port number:

    ```yaml
    http.port: 5067
    ```

2. Disable the default collection of Metricbeat monitoring metrics.<br>

    Add the following setting in the Metricbeat configuration file (`metricbeat.yml`):

    ```yaml
    monitoring.enabled: false
    ```

    For more information, see [Monitoring configuration options](/reference/metricbeat/configuration-monitor.md).

3. Configure host (optional).<br>

    If you intend to get metrics using {{metricbeat}} installed on another server, you need to bind the Metricbeat to host’s IP:

    ```yaml
    http.host: xxx.xxx.xxx.xxx
    ```

4. Configure cluster UUID.<br>

    The cluster UUID is necessary if you want to see {{beats}} monitoring in the {{kib}} stack monitoring view. The monitoring data will be grouped under the cluster for that UUID. To associate Metricbeat with the cluster UUID, set:

    ```yaml
    monitoring.cluster_uuid: "cluster-uuid"
    ```

5. Start Metricbeat.


## Install and configure {{metricbeat}} to collect monitoring data [configure-metricbeat]

1. The next step depends on how you want to run {{metricbeat}}:

    * If you’re running as a service and want to run a separate monitoring instance, take the the steps required for your environment to run two instances of {{metricbeat}} as a service. The steps for doing this vary by platform and are beyond the scope of this documentation.
    * If you’re running the binary directly in the foreground and want to run a separate monitoring instance, install {{metricbeat}} to a different path. If necessary, set `path.config`, `path.data`, and `path.log` to point to the correct directories. See [Directory layout](/reference/metricbeat/directory-layout.md) for the default locations.

2. Enable the `beat-xpack` module in {{metricbeat}}.<br>

    For example, to enable the default configuration in the `modules.d` directory, run the following command, using the correct command syntax for your OS:

    ```sh
    metricbeat modules enable beat-xpack
    ```

    For more information, see [Configure modules](/reference/metricbeat/configuration-metricbeat.md) and [beat module](/reference/metricbeat/metricbeat-module-beat.md).

3. Configure the `beat-xpack` module in {{metricbeat}}.<br>

    The `modules.d/beat-xpack.yml` file contains the following settings:

    ```yaml
    - module: beat
      metricsets:
        - stats
        - state
      period: 10s
      hosts: ["http://localhost:5066"]
      #username: "user"
      #password: "secret"
      xpack.enabled: true
    ```

    Set the `hosts`, `username`, and `password` settings as required by your environment. For other module settings, it’s recommended that you accept the defaults.

    By default, the module collects Metricbeat monitoring data from `localhost:5066`. If you exposed the metrics on a different host or port when you enabled the HTTP endpoint, update the `hosts` setting.

    To monitor multiple {{beats}} agents, specify a list of hosts, for example:

    ```yaml
    hosts: ["http://localhost:5066","http://localhost:5067","http://localhost:5068"]
    ```

    If you configured Metricbeat to use encrypted communications, you must access it via HTTPS. For example, use a `hosts` setting like `https://localhost:5066`.

    If the Elastic {{security-features}} are enabled, you must also provide a user ID and password so that {{metricbeat}} can collect metrics successfully:

    1. Create a user on the {{es}} cluster that has the `remote_monitoring_collector` [built-in role](elasticsearch://reference/elasticsearch/roles.md). Alternatively, if it’s available in your environment, use the `remote_monitoring_user` [built-in user](docs-content://deploy-manage/users-roles/cluster-or-deployment-auth/built-in-users.md).
    2. Add the `username` and `password` settings to the beat module configuration file.

4. Optional: Disable the system module in the {{metricbeat}}.

    By default, the [system module](/reference/metricbeat/metricbeat-module-system.md) is enabled. The information it collects, however, is not shown on the **Stack Monitoring** page in {{kib}}. Unless you want to use that information for other purposes, run the following command:

    ```sh
    metricbeat modules disable system
    ```

5. Identify where to send the monitoring data.<br>

    ::::{tip}
    In production environments, we strongly recommend using a separate cluster (referred to as the *monitoring cluster*) to store the data. Using a separate monitoring cluster prevents production cluster outages from impacting your ability to access your monitoring data. It also prevents monitoring activities from impacting the performance of your production cluster.
    ::::


    For example, specify the {{es}} output information in the {{metricbeat}} configuration file (`metricbeat.yml`):

    ```yaml
    output.elasticsearch:
      # Array of hosts to connect to.
      hosts: ["http://es-mon-1:9200", "http://es-mon2:9200"] <1>

      # Optional protocol and basic auth credentials.
      #protocol: "https"
      #api_key:  "id:api_key" <2>
      #username: "elastic"
      #password: "changeme"
    ```

    1. In this example, the data is stored on a monitoring cluster with nodes `es-mon-1` and `es-mon-2`.
    2. Specify one of `api_key` or `username`/`password`.


    If you configured the monitoring cluster to use encrypted communications, you must access it via HTTPS. For example, use a `hosts` setting like `https://es-mon-1:9200`.

    ::::{important}
    The {{es}} {{monitor-features}} use ingest pipelines. The cluster that stores the monitoring data must have at least one node with the `ingest` role.
    ::::


    If the {{es}} {{security-features}} are enabled on the monitoring cluster, you must provide a valid user ID and password so that {{metricbeat}} can send metrics successfully:

    1. Create a user on the monitoring cluster that has the `remote_monitoring_agent` [built-in role](elasticsearch://reference/elasticsearch/roles.md). Alternatively, if it’s available in your environment, use the `remote_monitoring_user` [built-in user](docs-content://deploy-manage/users-roles/cluster-or-deployment-auth/built-in-users.md).

        ::::{tip}
        If you’re using index lifecycle management, the remote monitoring user requires additional privileges to create and read indices. For more information, see [*Grant users access to secured resources*](/reference/metricbeat/feature-roles.md).
        ::::

    2. Add the `username` and `password` settings to the {{es}} output information in the {{metricbeat}} configuration file.

    For more information about these configuration options, see [Configure the {{es}} output](/reference/metricbeat/elasticsearch-output.md).

6. [Start {{metricbeat}}](/reference/metricbeat/metricbeat-starting.md) to begin collecting monitoring data.
7. [View the monitoring data in {{kib}}](docs-content://deploy-manage/monitor/stack-monitoring/kibana-monitoring-data.md).

