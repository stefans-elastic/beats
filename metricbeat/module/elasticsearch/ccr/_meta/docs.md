This is the `ccr` metricset of the {{es}} module. It uses the Cross-Cluster Replication Stats API endpoint to fetch metrics about cross-cluster replication from the {{es}} clusters that are participating in cross-cluster replication.

If the {{es}} cluster does not have cross-cluster replication enabled, this metricset will not collect metrics. A DEBUG log message about this will be emitted in the Metricbeat log.
