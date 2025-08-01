---
navigation_title: "dissect"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/dissect.html
applies_to:
  stack: ga
---

# Dissect strings [dissect]


The `dissect` processor tokenizes incoming strings using defined patterns.

```yaml
processors:
  - dissect:
      tokenizer: "%{key1} %{key2} %{key3|convert_datatype}"
      field: "message"
      target_prefix: "dissect"
```

The `dissect` processor has the following configuration settings:

`tokenizer`
:   The field used to define the **dissection** pattern. Optional convert datatype can be provided after the key using `|` as separator to convert the value from string to integer, long, float, double, boolean or ip.

`field`
:   (Optional) The event field to tokenize. Default is `message`.

`target_prefix`
:   (Optional) The name of the field where the values will be extracted. When an empty string is defined, the processor will create the keys at the root of the event. Default is `dissect`. When the target key already exists in the event, the processor won’t replace it and log an error; you need to either drop or rename the key before using dissect, or enable the `overwrite_keys` flag.

`ignore_failure`
:   (Optional) Flag to control whether the processor returns an error if the tokenizer fails to match the message field. If set to true, the processor will silently restore the original event, allowing execution of subsequent processors (if any). If set to false (default), the processor will log an error, preventing execution of other processors.

`overwrite_keys`
:   (Optional) When set to true, the processor will overwrite existing keys in the event. The default is false, which causes the processor to fail when a key already exists.

`trim_values`
:   (Optional) Enables the trimming of the extracted values. Useful to remove leading and/or trailing spaces. Possible values are:

    * `none`: (default) no trimming is performed.
    * `left`: values are trimmed on the left (leading).
    * `right`: values are trimmed on the right (trailing).
    * `all`: values are trimmed for leading and trailing.


`trim_chars`
:   (Optional) Set of characters to trim from values, when trimming is enabled. The default is to trim the space character (`" "`). To trim multiple characters, simply set it to a string containing all characters to trim. For example, `trim_chars: " \t"` will trim spaces and/or tabs.

For tokenization to be successful, all keys must be found and extracted, if one of them cannot be found an error will be logged and no modification is done on the original event.

::::{note}
A key can contain any characters except reserved suffix or prefix modifiers:  `/`,`&`, `+`, `#` and `?`.
::::


See [Conditions](/reference/heartbeat/defining-processors.md#conditions) for a list of supported conditions.

## Dissect example [dissect-example]

For this example, imagine that an application generates the following messages:

```sh
"321 - App01 - WebServer is starting"
"321 - App01 - WebServer is up and running"
"321 - App01 - WebServer is scaling 2 pods"
"789 - App02 - Database is will be restarted in 5 minutes"
"789 - App02 - Database is up and running"
"789 - App02 - Database is refreshing tables"
```

Use the `dissect` processor to split each message into three fields, for example, `service.pid`, `service.name` and `service.status`:

```yaml
processors:
  - dissect:
      tokenizer: '"%{service.pid|integer} - %{service.name} - %{service.status}"'
      field: "message"
      target_prefix: ""
```

This configuration produces fields like:

```json
"service": {
  "pid": 321,
  "name": "App01",
  "status": "WebServer is up and running"
},
```

`service.name` is an ECS [keyword field](elasticsearch://reference/elasticsearch/mapping-reference/keyword.md), which means that you can use it in {{es}} for filtering, sorting, and aggregations.

When possible, use ECS-compatible field names. For more information, see the [Elastic Common Schema](ecs://reference/index.md) documentation.


