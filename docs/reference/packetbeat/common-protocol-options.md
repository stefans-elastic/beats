---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/common-protocol-options.html
applies_to:
  stack: ga
---

# Common protocol options [common-protocol-options]

The following options are available for all protocols:


## `enabled` [_enabled_2]

The enabled setting is a boolean setting to enable or disable protocols without having to comment out configuration sections. If set to false, the protocol is disabled.

The default value is true.


## `ports` [_ports]

Exception: For ICMP the option `enabled` has to be used instead.

The ports where Packetbeat will look to capture traffic for specific protocols. Packetbeat installs a [BPF](https://en.wikipedia.org/wiki/Berkeley_Packet_Filter) filter based on the ports specified in this section. If a packet doesn’t match the filter, very little CPU is required to discard the packet. Packetbeat also uses the ports specified here to determine which parser to use for each packet.


## `send_request` [send-request-option]

If this option is enabled, the raw message of the request (`request` field) is sent to Elasticsearch. The default is false. This option is useful when you want to index the whole request. Note that for HTTP, the body is not included by default, only the HTTP headers.


## `send_response` [send-response-option]

If this option is enabled, the raw message of the response (`response` field) is sent to Elasticsearch. The default is false.  This option is useful when you want to index the whole response. Note that for HTTP, the body is not included by default, only the HTTP headers.


## `transaction_timeout` [transaction-timeout-option]

The per protocol transaction timeout. Expired transactions will no longer be correlated to incoming responses, but sent to Elasticsearch immediately.


## `fields` [packetbeat-configuration-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.


## `index` [packetbeat-configuration-index]

Overrides the index that events for the given protocol are published to.

```yaml
packetbeat.protocols:
- type: http
  ports: [80]
  fields:
    service_id: nginx
```


## `fields_under_root` [packetbeat-fields-under-root]

If this option is set to true, the custom [fields](#packetbeat-configuration-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Packetbeat, then the custom fields overwrite the other fields.


## `tags` [_tags_2]

A list of tags that will be sent with the transaction event. This setting is optional.


## `processors` [_processors_2]

A list of processors to apply to the data generated by the protocol.

See [Processors](/reference/packetbeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## `keep_null` [_keep_null_2]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.

