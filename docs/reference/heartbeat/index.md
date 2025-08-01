---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/heartbeat-overview.html
  - https://www.elastic.co/guide/en/beats/heartbeat/current/index.html
applies_to:
  stack: ga
---

# Heartbeat

Heartbeat is a lightweight daemon that you install on a remote server to periodically check the status of your services and determine whether they are available. Unlike [Metricbeat](/reference/metricbeat/index.md), which only tells you if your servers are up or down, Heartbeat tells you whether your services are reachable.

Heartbeat is useful when you need to verify that you’re meeting your service level agreements for service uptime. It’s also useful for other scenarios, such as security use cases, when you need to verify that no one from the outside can access services on your private enterprise server.

You can configure Heartbeat to ping all DNS-resolvable IP addresses for a specified hostname. That way, you can check all services that are load-balanced to see if they are available.

When you configure Heartbeat, you specify monitors that identify the hostnames that you want to check. Each monitor runs based on the schedule that you specify. For example, you can configure one monitor to run every 10 minutes, and a different monitor to run between the hours of 9:00 and 17:00.

Heartbeat currently supports monitors for checking hosts via:

* ICMP (v4 and v6) Echo Requests. Use the `icmp` monitor when you simply want to check whether a service is available. This monitor requires root access.
* TCP. Use the `tcp` monitor to connect via TCP. You can optionally configure this monitor to verify the endpoint by sending and/or receiving a custom payload.
* HTTP. Use the `http` monitor to connect via HTTP. You can optionally configure this monitor to verify that the service returns the expected response, such as a specific status code, response header, or content.

The `tcp` and `http` monitors both support SSL/TLS and some proxy settings.

Heartbeat is an Elastic [Beat](https://www.elastic.co/beats). It’s based on the `libbeat` framework. For more information, see the [Beats Platform Reference](/reference/index.md).

