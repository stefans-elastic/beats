---
navigation_title: "extract_array"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/extract-array.html
applies_to:
  stack: preview
---

# Extract array [extract-array]


::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


The `extract_array` processor populates fields with values read from an array field. The following example will populate `source.ip` with the first element of the `my_array` field, `destination.ip` with the second element, and `network.transport` with the third.

```yaml
processors:
  - extract_array:
      field: my_array
      mappings:
        source.ip: 0
        destination.ip: 1
        network.transport: 2
```

The following settings are supported:

`field`
:   The array field whose elements are to be extracted.

`mappings`
:   Maps each field name to an array index. Use 0 for the first element in the array. Multiple fields can be mapped to the same array element.

`ignore_missing`
:   (Optional) Whether to ignore events where the array field is missing. The default is `false`, which will fail processing of an event if the specified field does not exist. Set it to `true` to ignore this condition.

`overwrite_keys`
:   Whether the target fields specified in the mapping are overwritten if they already exist. The default is `false`, which will fail processing if a target field already exists.

`fail_on_error`
:   (Optional) If set to `true` and an error happens, changes to the event are reverted, and the original event is returned. If set to `false`, processing continues despite errors. Default is `true`.

`omit_empty`
:   (Optional) Whether empty values are extracted from the array. If set to `true`, instead of the target field being set to an empty value, it is left unset. The empty string (`""`), an empty array (`[]`) or an empty object (`{}`) are considered empty values. Default is `false`.

