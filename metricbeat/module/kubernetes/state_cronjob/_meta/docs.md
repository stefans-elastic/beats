This is the `state_cronjob` metricset of the Kubernetes module.

This metricset adds metadata by default only for versions of k8s >= v1.21. For older versions the APIs are not compatible and one need to configure the metricset with `add_metadata: false` and remove the proper `apiGroup` in the `ClusterRole`:

```yaml
- apiGroups: [ "batch" ]
  resources:
  - cronjobs
```
