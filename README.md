
## Project assembly

Created following the [Operator SDK Tutorial for Golang](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/).
```bash
operator-sdk init operator-sdk init --domain=open-cluster-management.io
operator-sdk create api --group=keycloak --version=v1alpha1 --kind=AuthorizationDomain
go mod vendor

make generate
make manifests
```
