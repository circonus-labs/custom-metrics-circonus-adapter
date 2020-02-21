# Custom Metrics - Circonus Adapter

Custom Metrics - Circonus Adapter is an implementation of [External Metrics API]
using Circonus SaaS as a backend. Its purpose is to enable pod autoscaling based
on Circonus CAQL statements

## Usage guide

This guide shows how to set up Custom Metrics - Circonus Adapter and export
metrics to Circonus in a compatible way. Once this is done, you can use
them to scale your application, following [HPA walkthrough].

### 1. Install the adapter

`kubectl apply -f https://raw.githubusercontent.com/circonus-labs/master/custom-metrics-circonus-adapter/deploy/production/adapter.yaml`

This creates a namespace: `custom-metrics` and all of the related perms and service account for the adapter.

### 2. Create your CAQL query config file

Autoscaling based on external metrics requires predefining all your queries in a config file to be passed to the custom metrics adapter as a config map.  The adapter scours the cluster for configmaps that contain a certain annotation: `circonus.com/k8s_custom_metrics_config` which should be set to the name of the ConfigMap field that contains the adapter configuration.  An example:

```sh
riley.berton(k8s: gke...st4_mlb-logs-npd-cluster1) $ cat test_config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-adapter-config
  namespace: my-apps-rule
  annotations:
    circonus.com/k8s_custom_metrics_config: "circonus_adapter_config"
data:
  circonus_adapter_config: |
    queries:
    - circonus_api_key: '12345678-1234-1234-1234-123456789012'
      caql: 'histogram:create{1,2,3,4,5}|histogram:mean()'
      external_name: histogram_mean
      window: 5m
      stride: 1m
  some_other_data_for_my_app: |
    some_field: foo
    some_other_field: bar
```

(Note that you would replace `circonus_api_key` with your actual api key)

When you `kubectl apply -f test_config.yaml` you will have an external metric called: `histogram_mean` in the `my-apps-rule` namespace:

```sh
 riley.berton(k8s: gke...st4_mlb-logs-npd-cluster1) $ kubectl get --raw '/apis/external.metrics.k8s.io/v1beta1' | jq
{
  "kind": "APIResourceList",
  "apiVersion": "v1",
  "groupVersion": "external.metrics.k8s.io/v1beta1",
  "resources": [
    {
      "name": "my-apps-rule/histogram_mean",
      "singularName": "",
      "namespaced": true,
      "kind": "ExternalMetricValueList",
      "verbs": [
        "get"
      ]
    }
  ]
```

And when we query this metric:

```sh
riley.berton(k8s: gke...st4_mlb-logs-npd-cluster1) $ kubectl get --raw '/apis/external.metrics.k8s.io/v1beta1/namespaces/my-apps-rule/histogram_mean' | jq
{
  "kind": "ExternalMetricValueList",
  "apiVersion": "external.metrics.k8s.io/v1beta1",
  "metadata": {
    "selfLink": "/apis/external.metrics.k8s.io/v1beta1/namespaces/my-apps-rule/histogram_mean"
  },
  "items": [
    {
      "metricName": "histogram_mean",
      "metricLabels": null,
      "timestamp": "2020-01-24T17:35:00Z",
      "value": "3048m"
    }
  ]
}
```

#### Configuration fields

* `queries` - (Required) List(Object): The list of query objects where each object is:

  * `circonus_api_key` - (Required) String: The api key so you can talk to your circonus account.
  * `caql` - (Required) String: The CAQL statement to execute, see [CAQL](https://login.circonus.com/resources/docs/user/CAQL.html)
  * `window` - (Optional) String: The amount of time going backwards from `time.Now()` that you want to query to create your metric,
  defaults to 5 minutes (5m).
  * `stride` - (Optional) String: The resolution of the response data.  Defaults to 1 minute (1m).
  * `aggregate` - (Optional) String: The function to use to combine all of the `stride`s in `window` into a single number since k8s is broken and if you return more than 1 datapoint it just adds them up!?  This defaults to `average` and must
  be one of: `average`, `min`, `max`
  
### 3. Configure your HPA

```yaml
apiVersion: autoscaling/v2beta1
kind: HorizontalPodAutoscaler
metadata:
  name: my-hpa
  namespace: my-namespace
spec:
  minReplicas: 1
  maxReplicas: 15
  scaleTargetRef:
    apiVersion: extensions/v1beta1
    kind: Deployment
    name: my-app
  metrics:
    - external:
        metricName: my_metric
        targetValue: 300
      type: External
```

The above will scale `my-app` in `my-namespace` using a metric called `my_metric` which would have been previously defined in a
config map as defined above.

### Metrics available from Circonus

Custom Metrics - Circonus Adapter exposes any time series or derived formula
which can be satisfied by the CAQL language.  The caveat is: if you execute
a CAQL query that returns more than 1 value per time period, this adapter will
use the first value returned.  So take care to compose CAQL that returns a single
value.

[Custom Metrics API]:
https://github.com/kubernetes/metrics/tree/master/pkg/apis/custom_metrics
[External Metrics API]:
https://github.com/kubernetes/metrics/tree/master/pkg/apis/external_metrics
[HPA walkthrough]:
https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale-walkthrough/
[cluster setup]: https://kubernetes.io/docs/setup/

### 4. Developing

#### Prerequisites

1. [Go](https://golang.org/) v1.13+
1. [Goreleaser](https://goreleaser.com/) v0.127.0+
1. [GolangCI-Lint](https://github.com/golangci/golangci-lint) v1.23.6+
