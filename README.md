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
2. Start *Custom Metrics - Circonus Adapter*.

  ```sh
  kubectl apply -f https://raw.githubusercontent.com/rileyberton/master/custom-metrics-circonus-adapter/deploy/production/adapter.yaml
  ```

3. Run a test query.

```sh
kubectl get --raw '/apis/external.metrics.k8s.io/v1beta1/namespaces/default/histogram:create{1,2,3,4,5}|histogram:mean()?labelSelector=circonus_api_key=<your api key>' | jq
{
  "kind": "ExternalMetricValueList",
  "apiVersion": "external.metrics.k8s.io/v1beta1",
  "metadata": {
    "selfLink": "/apis/external.metrics.k8s.io/v1beta1/namespaces/default/histogram:create%7B1,2,3,4,5%7D%7Chistogram:mean%28%29"
  },
  "items": [
    {
      "metricName": "histogram:create{1,2,3,4,5}|histogram:mean()",
      "metricLabels": null,
      "timestamp": "2019-11-25T20:53:00Z",
      "value": "3048m"
    }
  ]
}
```

Replace `<your api key>` above with your actual API key.

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
