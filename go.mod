module github.com/mdelder/multicluster-keycloak-operator

go 1.15

require (
	github.com/coreos/prometheus-operator v0.40.0 // indirect
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/google/go-jsonnet v0.16.0 // indirect
	github.com/keycloak/keycloak-operator v0.0.0-20210510074834-bfd2d9045f32
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/open-cluster-management/api v0.0.0-20210506014040-cc5aa104d1e1
	github.com/open-cluster-management/registration v0.0.0-20210127152631-4839ac6a02e2 // indirect
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/build-machinery-go v0.0.0-20210115170933-e575b44a7a94
	github.com/openshift/client-go v0.0.0-20201214125552-e615e336eb49 // indirect
	github.com/openshift/library-go v0.0.0-20201207213115-a0cd28f38065 // indirect
	github.com/operator-framework/operator-sdk v0.18.2 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.1
)

replace (
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.1
)
