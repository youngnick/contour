package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	projcontour "github.com/projectcontour/contour/apis/projectcontour/v1"
	"github.com/projectcontour/contour/internal/dag"
)

// HTTPProxyReconciler reconciles a HTTPProxy object
type HTTPProxyReconciler struct {
	client.Client
	Log logr.Logger
	ToController chan interface{}
}

// +kubebuilder:rbac:groups=networking.x.k8s.io,resources=HTTPProxyes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.x.k8s.io,resources=HTTPProxyes/status,verbs=get;update;patch

// Reconcile the changes.
func (r *HTTPProxyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("HTTPProxy", req.NamespacedName)

	// your logic here

	var proxy projcontour.HTTPProxy
	if err := r.Get(ctx, req.NamespacedName, &proxy); err != nil {
		notfound := client.IgnoreNotFound(err)
		if notfound != nil {
			log.Info(fmt.Sprintf("Unable to fetch HTTPProxy, %s", notfound))
			return ctrl.Result{}, notfound
		}
		log.Info("Issuing delete request")
		r.ToController <- dag.OpDelete{Obj: &proxy}
		return ctrl.Result{}, nil
	}

	log.Info("Issuing Upsert request")

	r.ToController <- dag.OpUpsert{Obj: &proxy}
	return ctrl.Result{}, nil
}

// SetupWithManager wires up the controller.
func (r *HTTPProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&projcontour.HTTPProxy{}).
		Complete(r)
}
