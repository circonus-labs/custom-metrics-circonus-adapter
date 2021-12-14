module github.com/circonus-labs/custom-metrics-circonus-adapter

go 1.13

require (
	github.com/circonus-labs/go-apiclient v0.7.15
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	google.golang.org/grpc v1.40.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/apiserver v0.23.0 // indirect
	k8s.io/client-go v0.23.0
	k8s.io/component-base v0.23.0
	k8s.io/klog v0.3.3
	k8s.io/metrics v0.23.0
	sigs.k8s.io/custom-metrics-apiserver v1.22.0
)
