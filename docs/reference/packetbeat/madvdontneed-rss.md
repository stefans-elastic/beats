---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/madvdontneed-rss.html
applies_to:
  stack: ga
---

# High RSS memory usage due to MADV settings [madvdontneed-rss]

In versions of Packetbeat prior to 7.10.2, the go runtime defaults to `MADV_FREE` by default. In some cases, this can lead to high RSS memory usage while the kernel waits to reclaim any pages assigned to Packetbeat. On versions prior to 7.10.2, set the `GODEBUG="madvdontneed=1"` environment variable if you run into RSS usage issues.

