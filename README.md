
## Overview

The multicluster-keycloak-operator is focused on providing a managed Single Sign-On (SSO) solution for a fleet of OpenShift clusters managed by Open Cluster Management.

The operator makes use of Keycloak to create a Github-backed identity provider.

1. For each `AuthorizationDomain`, a `KeycloakRealm` will be created to provide an identity provider for the hub and managed clusters.
2. For every ManagedCluster under management, a `KeycloakClient` will be created with a generated `clientId` and `clientSecret`.
3. From there, a `ManifestWork` will be created that injects the `clientSecret` into a `Secret` on the `ManagedCluster` and updates the `OAuth` `cluster` configuration to respect the `KeycloakRealm` identity provider hosted on the Hub cluster.

Currently, a work-in-progress.

## Getting Started

1. Install the Red Hat Single Sign-On Operator from the OpenShift Operator Catalog.
```
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  labels:
    operators.coreos.com/rhsso-operator.keycloak: ""
  name: rhsso-operator
  namespace: keycloak
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: rhsso-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  startingCSV: rhsso-operator.7.4.4
```
2. Configure a running instance of Keycloak, the service that is managed by the Red Hat SSO Operator.
```
apiVersion: keycloak.org/v1alpha1
kind: Keycloak
metadata:
  name: keycloak-sso
  labels:
    app: sso
  namespace: keycloak
spec:
  externalAccess:
    enabled: true
  instances: 1
```
3. Deploy the Multicluster Keycloak Operator into your cluster.
```
export KUBECONFIG=./path/to/your/kubeconfig
make install
make deploy
```

At this point, you can configure the `AuthorizationDomain` object that will reference your GitHub OAuthApp and inject configuration to each ManagedCluster to use the Red Hat Single Sign-On (SSO) operator that you configured above.

1. Register an OAuth App on GitHub (see the [GitHub docs](https://docs.github.com/en/developers/apps/creating-an-oauth-app)).
    - Use any value for the `homepageURL`.
    - Use `https://keycloak-keycloak.apps.<clusterName>.<baseDomain>/auth/realms/sso-ad/broker/github/endpoint` for the `OAuthCallbackURL`. The Route will be served by the instance of Keycloak deployed in step 3. The segment "sso-ad" is **MUST** be used as the name of the `AuthorizationDomain` created below. So if you create an `AuthorizationDomain` of a different name, then replace "sso-ad" in this URL with the name that you used for the `AuthorizationDomain`.
    - Retrieve the `clientID` and `clientSecret`.
2. Create a ConfigMap in the Namespace `keycloak` with the TLS Certificate for the route above (e.g. `https://*.apps.<clusterName>.<baseDomain>`):
```
apiVersion:
kind: ConfigMap
metadata:
  name: ca-config-map
  namespace: keycloak
data:
  ca.crt: |-
    -----BEGIN CERTIFICATE-----
   ...
    -----END CERTIFICATE-----
    -----BEGIN CERTIFICATE-----
    ...
    -----END CERTIFICATE-----
```
3. Create a Secret in the Namespace `keycloak` to store the GitHub OAuth `clientID` and `clientSecret`.
```
apiVersion: v1
kind: Secret
stringData:
  clientId: ...
  clientSecret: ...
metadata:
  name: github-oauth-credentials
  namespace: keycloak
type: Opaque
```
4. Create an `AuthorizationDomain`. The Multicluster Keycloak Operator (in this project) will reconcile the `AuthorizationDomain` CR and create the necessary `Keycloak` resources to allow any `ManagedCluster` to use the running `Keycloak` service on the Hub as an OAuth2 Identity Provider. In addition, every `ManagedCluster` will pick up a new `ManifestWork` that configures its local `OAuth` `cluster` CR.
```
apiVersion: keycloak.open-cluster-management.io/v1alpha1
kind: AuthorizationDomain
metadata:
  name: sso-ad
  namespace: keycloak
spec:
  identityProviders:
  - type: github
    secretRef: github-oauth-credentials
  issuerURL: "https://keycloak-keycloak.apps.<clusterName>.<baseDomain>/auth/realms/sso-ad"
  issuerCertificate:
    configMapRef: ca-config-map
```

**NOTE**: This project remains a work in progress and the various objects that are created may need to be removed manually if you delete the `AuthorizationDomain`. Be sure to have access to the TLS certificates or an alternative way to login to your cluster in case a problem occurs.

## Using Open Cluster Management policies to preconfigure RBAC on ManagedClusters.

The following two policies will allow you to precreate a Group backed by email addresses that will be retrieve from GitHub to create the relevant `Identity` and `User` object after a user accesses a `ManagedCluster` console.

```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  name: policy-group-demo-admins
  namespace: open-cluster-management-policies
  annotations:
    policy.open-cluster-management.io/standards: NIST-CSF
    policy.open-cluster-management.io/categories: PR.AC Identity Management Authentication and Access Control
    policy.open-cluster-management.io/controls: PR.AC-4 Access Control
spec:
  remediationAction: enforce
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: demo-admins-group-config-policy
        spec:
          severity: high
          object-templates:
            - complianceType: mustonlyhave
              objectDefinition:
                apiVersion: user.openshift.io/v1
                kind: Group
                metadata:
                  name: demo-admins
                users:
                - UPDATE THIS LIST WITH EMAIL ADDRESSES OF YOUR GITHUB USERS
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: demo-admins-clusterrolebinding-config-policy
        spec:
          severity: high
          object-templates:
            - complianceType: mustonlyhave # role definition should exact match
              objectDefinition:
                apiVersion: rbac.authorization.k8s.io/v1
                kind: ClusterRoleBinding
                metadata:
                  name: demo-admins
                roleRef:
                  apiGroup: rbac.authorization.k8s.io
                  kind: ClusterRole
                  name: cluster-admin
                subjects:
                - apiGroup: rbac.authorization.k8s.io
                  kind: Group
                  name: demo-admins
---
apiVersion: policy.open-cluster-management.io/v1
kind: PlacementBinding
metadata:
  name: binding-policy-group-demo-admins
  namespace: open-cluster-management-policies
placementRef:
  name: placement-policy-group-demo-admins
  kind: PlacementRule
  apiGroup: apps.open-cluster-management.io
subjects:
- name: policy-group-demo-admins
  kind: Policy
  apiGroup: policy.open-cluster-management.io
---
apiVersion: apps.open-cluster-management.io/v1
kind: PlacementRule
metadata:
  name: placement-policy-group-demo-admins
  namespace: open-cluster-management-policies
spec:
  clusterConditions:
  - status: "True"
    type: ManagedClusterConditionAvailable
  clusterSelector:
    matchExpressions: []

```

The following policy will remove the default `kubeadmin` credential from all `ManagedClusters`.
```
apiVersion: policy.open-cluster-management.io/v1
kind: Policy
metadata:
  name: no-kubeadmin-config-policy
  namespace: open-cluster-management-policies
  annotations:
    policy.open-cluster-management.io/standards: NIST-CSF
    policy.open-cluster-management.io/categories: PR.AC Identity Management Authentication and Access Control
    policy.open-cluster-management.io/controls: PR.AC-4 Access Control
spec:
  remediationAction: enforce
  disabled: false
  policy-templates:
    - objectDefinition:
        apiVersion: policy.open-cluster-management.io/v1
        kind: ConfigurationPolicy
        metadata:
          name: no-kubeadmin-config-policy
        spec:
          severity: high
          object-templates:
            - complianceType: mustnothave
              objectDefinition:
                apiVersion: v1
                kind: Secret
                metadata:
                  name: kubeadmin
                  namespace: kube-system
                type: Opaque
---
apiVersion: policy.open-cluster-management.io/v1
kind: PlacementBinding
metadata:
  name: binding-no-kubeadmin-config-policy
  namespace: open-cluster-management-policies
placementRef:
  name: placement-no-kubeadmin-config-policy
  kind: PlacementRule
  apiGroup: apps.open-cluster-management.io
subjects:
- name: no-kubeadmin-config-policy
  kind: Policy
  apiGroup: policy.open-cluster-management.io
---
apiVersion: apps.open-cluster-management.io/v1
kind: PlacementRule
metadata:
  name: placement-no-kubeadmin-config-policy
  namespace: open-cluster-management-policies
spec:
  clusterConditions:
  - status: "True"
    type: ManagedClusterConditionAvailable
  clusterSelector:
    matchExpressions: []
```


## Project assembly

The initial project was created following the [Operator SDK Tutorial for Golang](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/).

```bash
operator-sdk init operator-sdk init --domain=open-cluster-management.io
operator-sdk create api --group=keycloak --version=v1alpha1 --kind=AuthorizationDomain
go mod vendor

make generate
make manifests
```

## Developing/Contributing

Contributions are welcome and encouraged via Pull Requests.

```bash

make generate
make manifests
make install
make run
```

## References

[Operator SDK Advanced Topics](https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/)