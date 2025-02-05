/*
Copyright 2025 KubeSphere Authors

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

package main

import (
	"flag"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation"
	mutationtypes "github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	mutationsv1alpha1 "kubesphere.io/muato/api/mutations/v1alpha1"
	mutato "kubesphere.io/muato/pkg"
	"kubesphere.io/muato/pkg/controller"
	"kubesphere.io/muato/pkg/mutators"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

const eventQueueSize = 1024

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	runtime.Must(mutationsv1alpha1.AddToScheme(mgr.GetScheme()))

	mSys := mutation.NewSystem(mutation.SystemOpts{})
	events := make(chan event.GenericEvent, eventQueueSize)
	dynamic := controller.Adder{
		MutationSystem: mSys,
		Kind:           "Dynamic",
		NewMutationObj: func() client.Object { return &mutationsv1alpha1.Dynamic{} },
		MutatorFor: func(obj client.Object) (mutationtypes.Mutator, error) {
			dynamic := obj.(*mutationsv1alpha1.Dynamic)
			return mutators.MutatorForDynamic(dynamic)
		},
		Events: events,
	}
	if err := dynamic.Add(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Dynamic")
		os.Exit(1)
	}

	if err = (&mutato.Webhook{
		MutationSystem: mSys,
	}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Pod")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
