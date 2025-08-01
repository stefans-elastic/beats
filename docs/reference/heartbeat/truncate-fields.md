---
navigation_title: "truncate_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/truncate-fields.html
applies_to:
  stack: ga
---

# Truncate fields [truncate-fields]


The `truncate_fields` processor truncates a field to a given size. If the size of the field is smaller than the limit, the field is left as is.

`fields`
:   List of fields to truncate. It’s supported to use `@metadata.` prefix for the fields and truncate values in the event metadata instead of event fields.

`max_bytes`
:   Maximum number of bytes in a field. Mutually exclusive with `max_characters`.

`max_characters`
:   Maximum number of characters in a field. Mutually exclusive with `max_bytes`.

`fail_on_error`
:   (Optional) If set to true, in case of an error the changes to the event are reverted, and the original event is returned. If set to `false`, processing continues also if an error happens. Default is `true`.

`ignore_missing`
:   (Optional) Whether to ignore events that lack the source field. The default is `false`, which will fail processing of an event if a field is missing.

For example, this configuration truncates the field named `message` to 5 characters:

```yaml
processors:
  - truncate_fields:
      fields:
        - message
      max_characters: 5
      fail_on_error: false
      ignore_missing: true
```

