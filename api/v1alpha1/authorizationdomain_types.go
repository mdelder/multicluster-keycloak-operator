/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthorizationDomainSpec defines the desired state of AuthorizationDomain
type AuthorizationDomainSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Identify clusters that should be managed under this AuthorizationDomain
	// Placement *plrv1alpha1.Placement `json:"placement,omitempty"`

	// Identity providers allow you delegate the validation of the identity outside of Keycloak
	// examples include: {"type":"github", "secretRef": "<name of secret with clientId/clientSecret from GitHub OAuthApp>"}
	IdentityProviders []IdentityProvider `json:"identityProviders,omitempty"`

	// Identify Keycloak Issuer URL e.g. https://keycloak-keycloak.apps.foxtrot.demo.red-chesterfield.com/auth/realms/basic
	IssuerURL string `json:"issuerURL,omitempty"`
}

// AuthorizationDomainStatus defines the observed state of AuthorizationDomain
type AuthorizationDomainStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AuthorizationDomain is the Schema for the authorizationdomains API
type AuthorizationDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthorizationDomainSpec   `json:"spec,omitempty"`
	Status AuthorizationDomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AuthorizationDomainList contains a list of AuthorizationDomain
type AuthorizationDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthorizationDomain `json:"items"`
}

// Provide details to configure a IdentityProvider for the KeycloakRealm
type IdentityProvider struct {
	Type      string `json:"type,omitempty"`
	SecretRef string `json:"secretRef,omitempty"`
}

func init() {
	SchemeBuilder.Register(&AuthorizationDomain{}, &AuthorizationDomainList{})
}
