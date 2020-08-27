package route

import (
	"context"
	errs "errors"

	routev1 "github.com/openshift/api/route/v1"
	outils "github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "route_controller"
const finalizerName = "f5/cloudfirst.dev"

var log = logf.Log.WithName(controllerName)

// Add the controller to the manager
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRoute{
		ReconcilerBase: outils.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	reconcileRoute, ok := r.(*ReconcileRoute)
	if !ok {
		return errs.New("unable to convert to ReconcileRoute")
	}
	if ok, err := reconcileRoute.IsAPIResourceAvailable(schema.GroupVersionKind{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	}); !ok || err != nil {
		if err != nil {
			return err
		}
		return nil
	}

	// this will filter routes that have the annotation and on update only if the annotation is changed.
	isAnnotatedAndSecureRoute := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newRoute, ok := e.ObjectNew.DeepCopyObject().(*routev1.Route)
			if !ok {
				return false
			}
			oldRoute, _ := e.ObjectOld.DeepCopyObject().(*routev1.Route)
			if newRoute != nil {
				if (len(newRoute.Status.Ingress) > 0 && len(oldRoute.Status.Ingress) > 0 && oldRoute.Status.Ingress[0].RouterName != newRoute.Status.Ingress[0].RouterName) ||
					(len(newRoute.Status.Ingress) > 0 && len(oldRoute.Status.Ingress) == 0 && newRoute.Status.Ingress[0].RouterName != "") {
					return true
				}
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			route, ok := e.Object.DeepCopyObject().(*routev1.Route)
			if !ok {
				return false
			}
			if len(route.Status.Ingress) > 0 && route.Status.Ingress[0].Conditions[0].Type == routev1.RouteAdmitted &&
				route.Status.Ingress[0].Conditions[0].Status == corev1.ConditionTrue {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},

		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	// Watch for changes to primary resource Route
	err = c.Watch(&source.Kind{Type: &routev1.Route{
		TypeMeta: v1.TypeMeta{
			Kind: "Route",
		},
	}}, &handler.EnqueueRequestForObject{}, isAnnotatedAndSecureRoute)

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRoute{}

// ReconcileRoute reconciles a Route object
type ReconcileRoute struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	outils.ReconcilerBase
}

// Reconcile the route
func (r *ReconcileRoute) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Route")

	// Fetch the Route instance
	instance := &routev1.Route{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if outils.IsBeingDeleted(instance) {
		reqLogger.Info("going to be deleted")
		outils.RemoveFinalizer(instance, finalizerName)

		// remove f5 entry
		r.removeF5Entry(instance)
	} else {
		reqLogger.Info("Addining finalizer to Route")
		outils.AddFinalizer(instance, finalizerName)

		// add f5 entry
		r.addF5Entry(instance)
	}

	// Update route with any changes that were made
	err = r.GetClient().Update(context.TODO(), instance)
	if err != nil {
		log.Error(err, "unable to update instance", "instance", instance)
		return r.ManageError(instance, err)
	}

	// if we are here we know it's because a route was create/modified or its referenced secret was created/modified
	// therefore the only think we need to do is to update the route certificates

	return r.ManageSuccess(instance)
}

func (r *ReconcileRoute) addF5Entry(instance *routev1.Route) error {
	return nil
}

func (r *ReconcileRoute) removeF5Entry(instance *routev1.Route) error {
	return nil
}
