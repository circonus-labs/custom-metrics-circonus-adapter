# Custom Metrics - Circonus Adapter

Custom Metrics - Circonus Adapter is an implementation of [External Metrics API]
using Circonus SaaS as a backend. Its purpose is to enable pod autoscaling based
on Circonus CAQL statements

This is mostly copied from: https://github.com/GoogleCloudPlatform/k8s-stackdriver/tree/master/custom-metrics-stackdriver-adapter

## Usage guide

This guide shows how to set up Custom Metrics - Circonus Adapter and export
metrics to Circonus in a compatible way. Once this is done, you can use
them to scale your application, following [HPA walkthrough].

### Configure cluster

1. Create Kubernetes cluster or use existing one, see [cluster setup].
   Requirements:

   * Kubernetes version 1.8.1 or newer running on GKE or GCE

   * Monitoring scope `monitoring` set up on cluster nodes. **It is enabled by
     default, so you should not have to do anything**. See also [OAuth 2.0 API
     Scopes] to learn more about authentication scopes.

     You can use following commands to verify that the scopes are set correctly:
     - For GKE cluster `<my_cluster>`, use following command:
       ```
       gcloud container clusters describe <my_cluster>
       ```
       For each node pool check the section `oauthScopes` - there should be
       `https://www.googleapis.com/auth/monitoring` scope listed there.
     - For a GCE instance `<my_instance>` use following command:
       ```
       gcloud compute instances describe <my_instance>
       ```
       `https://www.googleapis.com/auth/monitoring` should be listed in the
       `scopes` section.


     To configure set scopes manually, you can use:
     - `--scopes` flag if you are using `gcloud container clusters create`
       command, see [gcloud
       documentation](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create).
     - Environment variable `NODE_SCOPES` if you are using [kube-up.sh script].
       It is enabled by default.
     - To set scopes in existing clusters you can use `gcloud beta compute
       instances set-scopes` command, see [gcloud
       documentation](https://cloud.google.com/sdk/gcloud/reference/beta/compute/instances/set-scopes).
    * On GKE, you need cluster-admin permissions on your cluster. You can grant
      your user account these permissions with following command:
      ```
      kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $(gcloud config get-value account)
      ```
2. Create your autoscaling CAQL query config file

Autoscaling based on external metrics requires predefinig all your queries in a config file to be passed to the 
custom metrics adapter when it starts.  This is due to a limitation in how the external metric is defined
in the HPA config: there is no reasonable way to send a complex query in the external metric config and the 
recommended way for now is to define these complex queries in your custom metrics adapater config and expose the
results with simple names that can be used by the HPA for scaling.  This approach closely follows the 
prometheus k8s adapter.

To create your query config map you can follow the `deploy/production/query_config_map_template.yaml` file and edit
it to include the CAQL statements and names you require for your own autoscaling needs.  A config map might resemble:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: adapter-config
  namespace: custom-metrics
data:
  config.yaml: |
    queries:
    - circonus_api_key: '12345678-1234-1234-1234-123456789012'
      caql: 'histogram:create{1,2,3,4,5}|histogram:mean()'
      external_name: histogram_mean
      window: 5m
      stride: 1m
```

`queries` is a list of queries.  Each can be associated with a different API key in case you need to pull data
from more than 1 circonus account.

  `circonus_api_key` is the API Token that is associated with the circonus account you want to query data from.  
  Importantly, you cannot query data across accounts with this adapter.

  `caql` is the CAQL statement you wish to execute.  It's important to note that the CAQL statement must produce a 
  single number per time period (stride).  If you execute a CAQL statement that produces multiple streams of data 
  at each stride, this adapter will use the first stream in the results so take care.  Also, returning histogram 
  data will notwork at all, you have to aggregate the data down to a single number if you have a distribution.

  `external_name` is the name you want to call the results.  This is the name you would refer to in your horizontal pod
  autoscaler configuration.  This must be unique across all the queries in your configuration.  The adapter will error
  out if there are duplicate names in your config.

  `window` is the time window of data to fetch.  Often metrics that run up against the edge of `now` are incomplete. 
  `window` allows you to fetch more than 1 point in time.  If you use `5m` for `window` with a `stride` of `1m`, it 
  will return the last 5 minutes of data with 1 minute granularity.  This defaults to `5m` (300 seconds).

  `stride` the granularity (or period) of the data returned.  This defaults to `1m` (60 seconds).

3. Start *Custom Metrics - Circonus Adapter*.

  ```sh
  kubectl apply -f https://raw.githubusercontent.com/rileyberton/master/custom-metrics-circonus-adapter/deploy/production/adapter.yaml -f your_query_config_map.yaml
  ```

4. Run a test query.

```sh
kubectl get --raw '/apis/external.metrics.k8s.io/v1beta1/namespaces/default/histogram_mean' | jq
{
  "kind": "ExternalMetricValueList",
  "apiVersion": "external.metrics.k8s.io/v1beta1",
  "metadata": {
    "selfLink": "/apis/external.metrics.k8s.io/v1beta1/namespaces/default/histogram_mean"
  },
  "items": [
    {
      "metricName": "histogram_mean",
      "metricLabels": null,
      "timestamp": "2019-11-25T20:53:00Z",
      "value": "3048m"
    }
    ...
  ]
}
```

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
