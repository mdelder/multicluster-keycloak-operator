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

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	keycloakv1alpha1 "github.com/mdelder/multicluster-keycloak-operator/api/v1alpha1"
	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ocmworkv1 "github.com/open-cluster-management/api/work/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

const keycloakNS = "keycloak"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("..", "vendor", "github.com", "open-cluster-management", "api", "cluster", "v1"),
			filepath.Join("..", "vendor", "github.com", "open-cluster-management", "api", "work", "v1"),
			filepath.Join("..", "test", "keycloakcrds"),
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = keycloakv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = keycloak.SchemeBuilder.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = ocmclusterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = ocmworkv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	err = (&AuthorizationDomainReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AuthorizationDomain"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&ManagedClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ManagedCluster"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	stopCh := ctrl.SetupSignalHandler()
	go func() {
		err = mgr.Start(stopCh)
		fmt.Printf("failed to starrt manager, %v\n", err)
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = mgr.GetClient()
	Expect(k8sClient).NotTo(BeNil())

	prepareKeyCloakNS()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func prepareKeyCloakNS() {
	keycloakNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "keycloak",
		},
	}
	err := k8sClient.Create(context.Background(), keycloakNS)
	Expect(err).ToNot(HaveOccurred())
}
