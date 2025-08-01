---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/linux-seccomp.html
applies_to:
  stack: beta
---

# Use Linux Secure Computing Mode (seccomp) [linux-seccomp]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


On Linux 3.17 and later, Metricbeat can take advantage of secure computing mode, also known as seccomp. Seccomp restricts the system calls that a process can issue. Specifically Metricbeat can load a seccomp BPF filter at process start-up that drops the privileges to invoke specific system calls. Once a filter is loaded by the process it cannot be removed.

The kernel exposes a large number of system calls that are not used by Metricbeat. By installing a seccomp filter, you can limit the total kernel surface exposed to Metricbeat (principle of least privilege). This minimizes the impact of unknown vulnerabilities that might be found in the process.

The filter is expressed as a Berkeley Packet Filter (BPF) program. The BPF program is generated based on a policy defined by Metricbeat. The policy can be customized through configuration as well.

A seccomp policy is architecture specific due to the fact that system calls vary by architecture. Metricbeat includes a whitelist seccomp policy for the amd64 and 386 architectures. You can view those policies [here](https://github.com/elastic/beats/tree/master/libbeat/common/seccomp).


## Seccomp Policy Configuration [seccomp-policy-config]

The seccomp policy can be customized through the configuration policy. This is an example blacklist policy that prohibits `execve`, `execveat`, `fork`, and `vfork` syscalls.

```yaml
seccomp:
  default_action: allow <1>
  syscalls:
  - action: errno <2>
    names: <3>
    - execve
    - execveat
    - fork
    - vfork
```

1. If the system call being invoked by the process does not match one of the names below then it will be allowed.
2. If the system call being invoked matches one of the names below then an error will be returned to caller. This is known as a blacklist policy.
3. These are system calls being prohibited.


These are the configuration options for a seccomp policy.

**`enabled`**
:   On Linux, this option is enabled by default. To disable seccomp filter loading, set this option to `false`.

**`default_action`**
:   The default action to take when none of the defined system calls match. See [action](#seccomp-policy-config-action) for the full list of values. This is required.

**`syscalls`**
:   Each object in this list must contain an `action` and a list of system call `names`. The list must contain at least one item.

**`names`**
:   A list of system call names. The system call name must exist for the runtime architecture, otherwise an error will be logged and the filter will not be installed. At least one system call must be defined.

$$$seccomp-policy-config-action$$$

**`action`**
:   The action to take when any of the system calls listed in `names` is executed. This is required. These are the available action values. The actions that are available depend on the kernel version.

    * `errno` - The system call will return `EPERM` (permission denied) to the caller.
    * `trace` - The kernel will notify a `ptrace` tracer. If no tracer is present then the system call fails with `ENOSYS` (function not implemented).
    * `trap` - The kernel will send a `SIGSYS` signal to the calling thread and not execute the system call. The Go runtime will exit.
    * `kill_thread` - The kernel will immediately terminate the thread. Other threads will continue to execute.
    * `kill_process` - The kernel will terminate the process. Available in Linux 4.14 and later.
    * `log` - The kernel will log the system call before executing it. Available in Linux 4.14 and later. (This does not go to the Beat’s log.)
    * `allow` - The kernel will allow the system call to execute.



## Auditbeat Reports Seccomp Violations [_auditbeat_reports_seccomp_violations]

You can use Auditbeat to report any seccomp violations that occur on the system. The kernel generates an event for each violation and Auditbeat reports the event. The `event.action` value will be `violated-seccomp-policy` and the event will contain information about the process and system call.

