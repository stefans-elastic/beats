---
navigation_title: "community_id"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/community-id.html
applies_to:
  stack: ga
---

# Community ID Network Flow Hash [community-id]


The `community_id` processor computes a network flow hash according to the [Community ID Flow Hash specification](https://github.com/corelight/community-id-spec).

The flow hash is useful for correlating all network events related to a single flow. For example you can filter on a community ID value and you might get back the Netflow records from multiple collectors and layer 7 protocol records from Packetbeat.

By default the processor is configured to read the flow parameters from the appropriate Elastic Common Schema (ECS) fields. If you are processing ECS data then no parameters are required.

```yaml
processors:
  - community_id:
```

If the data does not conform to ECS then you can customize the field names that the processor reads from. You can also change the `target` field which is where the computed hash is written to.

```yaml
processors:
  - community_id:
      fields:
        source_ip: my_source_ip
        source_port: my_source_port
        destination_ip: my_dest_ip
        destination_port: my_dest_port
        iana_number: my_iana_number
        transport: my_transport
        icmp_type: my_icmp_type
        icmp_code: my_icmp_code
      target: network.community_id
```

If the necessary fields are not present in the event then the processor will silently continue without adding the target field.

The processor also accepts an optional `seed` parameter that must be a 16-bit unsigned integer. This value gets incorporated into all generated hashes.

