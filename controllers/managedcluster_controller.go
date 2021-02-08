package controllers

import (
	"context"

	"github.com/go-logr/logr"
	multiclusterkeycloakv1alpha1 "github.com/mdelder/multicluster-keycloak-operator/api/v1alpha1"
	ocmclusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagedClusterReconciler reconciles a ManagedCluster object
type ManagedClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ocmclusterv1.ManagedCluster{}).
		Complete(r)
}

func (r *ManagedClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("managedcluster", req.NamespacedName)
	cluster := &ocmclusterv1.ManagedCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}
	r.Log.Info("Reconciling", "ManagedClusterr", cluster)

	authzDomainList := &multiclusterkeycloakv1alpha1.AuthorizationDomainList{}
	if err := r.Client.List(context.Background(), authzDomainList); err != nil {
		return ctrl.Result{}, err
	}

	errs := []error{}
	for _, authzDomain := range authzDomainList.Items {
		r.Log.Info("Discovered ManagedCluster", "ManagedClusterName", cluster.Name)

		clusterContext, err := newManagedClusterSSOContext(r.Client, &authzDomain, cluster, baseDomain)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := clusterContext.createOrUpdateKeycloakClient(); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := clusterContext.createOrUpdateManifestWork(); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		return ctrl.Result{}, utilerrors.NewAggregate(errs)
	}
	return ctrl.Result{}, nil
}
