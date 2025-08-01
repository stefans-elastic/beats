---
navigation_title: "add_host_metadata"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/add-host-metadata.html
applies_to:
  stack: ga
---

# Add Host metadata [add-host-metadata]


```yaml
processors:
  - add_host_metadata:
      cache.ttl: 5m
      geo:
        name: nyc-dc1-rack1
        location: 40.7128, -74.0060
        continent_name: North America
        country_iso_code: US
        region_name: New York
        region_iso_code: NY
        city_name: New York
```

It has the following settings:

`netinfo.enabled`
:   (Optional) Default true. Include IP addresses and MAC addresses as fields host.ip and host.mac

`cache.ttl`
:   (Optional) The processor uses an internal cache for the host metadata. This sets the cache expiration time. The default is 5m, negative values disable caching altogether.

`geo.name`
:   (Optional) User definable token to be used for identifying a discrete location. Frequently a datacenter, rack, or similar.

`geo.location`
:   (Optional) Longitude and latitude in comma separated format.

`geo.continent_name`
:   (Optional) Name of the continent.

`geo.country_name`
:   (Optional) Name of the country.

`geo.region_name`
:   (Optional) Name of the region.

`geo.city_name`
:   (Optional) Name of the city.

`geo.country_iso_code`
:   (Optional) ISO country code.

`geo.region_iso_code`
:   (Optional) ISO region code.

`replace_fields`
:   (Optional) Default true. If set to false, original host fields from the event will not be replaced by host fields from `add_host_metadata`.

The `add_host_metadata` processor annotates each event with relevant metadata from the host machine. The fields added to the event look like the following:

```json
{
   "host":{
      "architecture":"x86_64",
      "name":"example-host",
      "id":"",
      "os":{
         "family":"darwin",
         "type":"macos",
         "build":"16G1212",
         "platform":"darwin",
         "version":"10.12.6",
         "kernel":"16.7.0",
         "name":"Mac OS X"
      },
      "ip": ["192.168.0.1", "10.0.0.1"],
      "mac": ["00:25:96:12:34:56", "72:00:06:ff:79:f1"],
      "geo": {
          "continent_name": "North America",
          "country_iso_code": "US",
          "region_name": "New York",
          "region_iso_code": "NY",
          "city_name": "New York",
          "name": "nyc-dc1-rack1",
          "location": "40.7128, -74.0060"
        }
   }
}
```

Note: `add_host_metadata` processor will overwrite host fields if `host.*` fields already exist in the event from Beats by default with `replace_fields` equals to `true`. Please use `add_observer_metadata` if the beat is being used to monitor external systems.

