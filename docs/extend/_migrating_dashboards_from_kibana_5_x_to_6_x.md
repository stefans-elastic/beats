---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/_migrating_dashboards_from_kibana_5_x_to_6_x.html
---

# Migrating dashboards from Kibana 5.x to 6.x [_migrating_dashboards_from_kibana_5_x_to_6_x]

This section is useful for the community Beats to migrate the Kibana 5.x dashboards to 6.x dashboards.

In the Kibana 5.x, the saved dashboards consist of multiple JSON files, one for each dashboard, search, visualization and index-pattern. To import a dashboard in Kibana, you need to load not only the JSON file containing the dashboard, but also all its dependencies (searches, visualizations).

Starting with Kibana 6.0, the dashboards are loaded by default via the Kibana API. In this case, the saved dashboard consist of a single JSON file that includes not only the dashboard content, but also all its dependencies.

As the format of the dashboards and index-pattern for Kibana 5.x is different than the ones for Kibana 6.x, they are placed in different directories. Depending on the Kibana version, the 5.x or 6.x dashboards are loaded.

The Kibana 5.x dashboards are placed under the 5.x directory that contains the following directories:
- search
- visualization
- dashboard
- index-pattern

The Kibana 6.x dashboards and later are placed under the default directory that contains the following directories:
- dashboard
- index-pattern

:::{note}
Please make sure the 5.x and default directories are created before running the following commands.
:::

To migrate your Kibana 5.x dashboards to Kibana 6.0 and above, you can import the dashboards into Kibana 5.6 and then export them using Beats 6.0 version.

* Start Kibana 5.6
* Import Kibana 5.x dashboards using Beats 6.0 version.

Before importing the dashboards, make sure you run `make update` in the Beat directory, that updates the `_meta/kibana` directory. It generates the index-pattern from the `fields.yml` file, and places it under the `5.x/index-pattern` and `default/index-pattern` directories. In case of Metricbeat, Filebeat and Auditbeat, it collects the dashboards from all the modules to the `_meta/kibana` directory.

```shell
make update
```

Then load all the Beat’s dashboards. For example, to load the Metricbeat rabbitmq dashboards together with the Metricbeat index-pattern into Kibana 5.6, using the Kibana API:

```shell
make update
./metricbeat setup -E setup.dashboards.directory=_meta/kibana
```

* Export the dashboards using Beats 6.0 version.

You can export the dashboards via the Kibana API by using the [export_dashboards.go](https://github.com/elastic/beats/blob/main/dev-tools/cmd/dashboards/export_dashboards.go) application.

For example, to export the Metricbeat rabbitmq dashboard:

```shell
cd beats/metricbeat
go run ../dev-tools/cmd/dashboards/export_dashboards.go -dashboards Metricbeat-Rabbitmq -output
module/rabbitmq/_meta/kibana/default/Metricbeat-Rabbitmq.json <1>
```

1. `Metricbeat-Rabbitmq` is the ID of the dashboard that you want to export.


:::{note}
You can get the dashboard ID from the URL of the dashboard in Kibana. Depending on the Kibana version the dashboard was created, the ID consists of a name or random characters that can be separated by `-`.
:::

This command creates a single JSON file (Metricbeat-Rabbitmq.JSON) that contains the dashboard and all the dependencies like searches, visualizations. The name of the output file has the format: <Beat name>-<module name>.json.

Starting with Beats 6.0.0, you can create an `yml` file for each module or for the entire Beat with all the dashboards. Below is an example of the `module.yml` file for the system module in Metricbeat.

```yaml
dashboards:
    - id: Metricbeat-system-overview <1>
      file: Metricbeat-system-overview.json <2>

    - id: 79ffd6e0-faa0-11e6-947f-177f697178b8
      file: Metricbeat-host-overview.json

    - id: CPU-slash-Memory-per-container
      file: Metricbeat-docker-overview.json
```

1. Dashboard ID.
2. The JSON file where the dashboard is saved on disk.


Using the yml file, you can export all the dashboards for a single module or for the entire Beat using a single command:

```shell
cd metricbeat/module/system
go run ../../../dev-tools/cmd/dashboards/export_dashboards.go -yml module.yml
```

