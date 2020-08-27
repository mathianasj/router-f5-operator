module github.com/mathianasj/router-f5-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/prometheus/common v0.9.1
	github.com/redhat-cop/cert-utils-operator v0.2.1
	github.com/redhat-cop/operator-utils v0.3.4
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
)

replace k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
