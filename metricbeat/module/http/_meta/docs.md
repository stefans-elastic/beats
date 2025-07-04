The HTTP module is a Metricbeat module used to call arbitrary HTTP endpoints for which a dedicated Metricbeat module is not available.

Multiple endpoints can be configured which are polled in a regular interval and the result is shipped to the configured output channel. It is recommended to install a Metricbeat instance on each host from which data should be fetched.

This module is inspired by the Logstash [http_poller](logstash-docs-md://lsr/plugins-inputs-http_poller.md) input filter but doesn’t require that the endpoint is reachable by Logstash as the Metricbeat module pushes the data to the configured output channels, e.g. Logstash or Elasticsearch.

This is often necessary in security restricted network setups, where Logstash is not able to reach all servers. Instead the server to be monitored itself has Metricbeat installed and can send the data or a collector server has Metricbeat installed which is deployed in the secured network environment and can reach all servers to be monitored.

::::{note}
As the HTTP metricsets also fetch headers, this can lead to lots of fields in Elasticsearch in case there are many different headers. If this is the case for you and you don’t need the headers, we recommend to use processors to filter out the header field.
::::
