module github.com/mdelder/multicluster-keycloak-operator

go 1.15

require (
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/keycloak/keycloak-operator v0.0.0-20210127091446-48fe7d86eabe
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/open-cluster-management/api v0.0.0-20201126023000-353dd8370f4d
	github.com/open-cluster-management/registration v0.0.0-20210127152631-4839ac6a02e2 // indirect
	github.com/openshift/build-machinery-go v0.0.0-20210115170933-e575b44a7a94
	github.com/openshift/library-go v0.0.0-20210127081712-a4f002827e42 // indirect
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
