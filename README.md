
## Overview

The multicluster-keycloak-operator is focused on providing a managed Single Sign-On (SSO) solution for a fleet of OpenShift clusters managed by Open Cluster Management.

The operator makes use of Keycloak to create a Github-backed identity provider.

1. For each `AuthorizationDomain`, a `KeycloakRealm` will be created to provide an identity provider for the hub and managed clusters.
2. For every ManagedCluster under management, a `KeycloakClient` will be created with a generated `clientId` and `clientSecret`.
3. From there, a `ManifestWork` will be created that injects the `clientSecret` into a `Secret` on the `ManagedCluster` and updates the `OAuth` `cluster` configuration to respect the `KeycloakRealm` identity provider hosted on the Hub cluster.

Currently, a work-in-progress.


## Project assembly

Created following the [Operator SDK Tutorial for Golang](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/).
```bash
operator-sdk init operator-sdk init --domain=open-cluster-management.io
operator-sdk create api --group=keycloak --version=v1alpha1 --kind=AuthorizationDomain
go mod vendor

make generate
make manifests
```

## References

[Operator SDK Advanced Topics](https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/)