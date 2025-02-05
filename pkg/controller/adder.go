package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/types"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Adder struct {
	// MutationSystem holds a reference to the mutation system to which
	// mutators will be registered/deregistered
	MutationSystem *mutation.System
	// Kind for the mutation object that is being reconciled
	Kind string
	// NewMutationObj creates a new instance of a mutation struct that can
	// be fed to the API server client for Get/Delete/Update requests
	NewMutationObj func() client.Object
	// MutatorFor takes the object returned by NewMutationObject and
	// turns it into a mutator. The contents of the mutation object
	// are set by the API server.
	MutatorFor func(client.Object) (types.Mutator, error)
	// Events enables queueing other Mutators for updates.
	Events chan event.GenericEvent
	// EventsSource watches for events broadcast to Events.
	// If multiple controllers listen to EventsSource, then
	// each controller gets a copy of each event.
	EventsSource source.Source
}

// Add creates a new Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func (a *Adder) Add(mgr manager.Manager) error {
	r := newReconciler(mgr, a.MutationSystem, a.Kind, a.NewMutationObj, a.MutatorFor, a.Events)
	return a.add(mgr, r)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func (a *Adder) add(mgr manager.Manager, r *Reconciler) error {
	// Create a new controller
	c, err := controller.New(fmt.Sprintf("%s-controller", strings.ToLower(r.gvk.Kind)), mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Mutators.
	err = c.Watch(
		source.Kind(mgr.GetCache(), r.newMutationObj(),
			&handler.EnqueueRequestForObject{}))
	if err != nil {
		return err
	}

	if a.EventsSource != nil {
		// Watch for enqueued events.
		err = c.Watch(
			source.Channel(a.Events,
				handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
					if obj.GetObjectKind().GroupVersionKind().Kind != r.gvk.Kind {
						return nil
					}
					return []reconcile.Request{{
						NamespacedName: apitypes.NamespacedName{
							Namespace: obj.GetNamespace(),
							Name:      obj.GetName(),
						},
					}}
				})))
	}

	return err
}
