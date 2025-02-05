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

package pkg

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation"
	mutationtypes "github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/types"
	"github.com/open-policy-agent/gatekeeper/v3/pkg/util"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"time"
)

const (
	namespaceKind      = "Namespace"
	serviceAccountName = "mutato"
)

type requestResponse string

const (
	successResponse requestResponse = "success"
	unknownResponse requestResponse = "unknown"
	skipResponse    requestResponse = "skip"
)

type Webhook struct {
	logger         logr.Logger
	client         client.Client
	reader         client.Reader
	decoder        runtime.Decoder
	MutationSystem *mutation.System
}

var (
	serviceaccount = fmt.Sprintf("system:serviceaccount:%s:%s", util.GetNamespace(), serviceAccountName)
)

func (r *Webhook) mutateRequest(ctx context.Context, req *admission.Request) admission.Response {
	ns := &corev1.Namespace{}

	// if the object being mutated is a namespace itself, we use it as namespace
	switch {
	case req.Kind.Kind == namespaceKind && req.Kind.Group == "":
		req.Namespace = ""
		obj, _, err := r.decoder.Decode(req.Object.Raw, nil, &corev1.Namespace{})
		if err != nil {
			return admission.Errored(int32(http.StatusInternalServerError), err)
		}
		ok := false
		ns, ok = obj.(*corev1.Namespace)
		if !ok {
			return admission.Errored(int32(http.StatusInternalServerError), errors.New("failed to cast namespace object"))
		}
	case req.AdmissionRequest.Namespace != "":
		if err := r.client.Get(ctx, types.NamespacedName{Name: req.AdmissionRequest.Namespace}, ns); err != nil {
			if !apierrors.IsNotFound(err) {
				r.logger.Error(err, "error retrieving namespace", "name", req.AdmissionRequest.Namespace)
				return admission.Errored(int32(http.StatusInternalServerError), err)
			}
			// bypass cached client and ask api-server directly
			err = r.reader.Get(ctx, types.NamespacedName{Name: req.AdmissionRequest.Namespace}, ns)
			if err != nil {
				r.logger.Error(err, "error retrieving namespace from API server", "name", req.AdmissionRequest.Namespace)
				return admission.Errored(int32(http.StatusInternalServerError), err)
			}
		}
	default:
		ns = nil
	}
	obj := unstructured.Unstructured{}
	err := obj.UnmarshalJSON(req.Object.Raw)
	if err != nil {
		r.logger.Error(err, "failed to unmarshal", "object", string(req.Object.Raw))
		return admission.Errored(int32(http.StatusInternalServerError), err)
	}

	// It is possible for the namespace to not be populated on an object.
	// Assign the namespace from the request object (which will have the appropriate
	// value), then restore the original value at the end to avoid sending a namespace patcr.
	oldNS := obj.GetNamespace()
	obj.SetNamespace(req.Namespace)

	mutable := &mutationtypes.Mutable{
		Object:    &obj,
		Namespace: ns,
		Username:  req.AdmissionRequest.UserInfo.Username,
		Source:    mutationtypes.SourceTypeOriginal,
	}

	mutated, err := r.MutationSystem.Mutate(mutable)
	if err != nil {
		r.logger.Error(err, "failed to mutate object", "object", string(req.Object.Raw))
		return admission.Errored(int32(http.StatusInternalServerError), err)
	}
	if !mutated {
		return admission.Allowed("Resource was not mutated")
	}

	mutable.Object.SetNamespace(oldNS)
	newJSON, err := mutable.Object.MarshalJSON()
	if err != nil {
		r.logger.Error(err, "failed to marshal mutated object", "object", obj)
		return admission.Errored(int32(http.StatusInternalServerError), err)
	}
	resp := admission.PatchResponseFromRaw(req.Object.Raw, newJSON)
	return resp
}

func isMutatoServiceAccount(user authenticationv1.UserInfo) bool {
	return user.Username == serviceaccount
}

func (r *Webhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	timeStart := time.Now()

	if isMutatoServiceAccount(req.AdmissionRequest.UserInfo) {
		return admission.Allowed("Mutato does not self-manage")
	}

	if req.AdmissionRequest.Operation != admissionv1.Create &&
		req.AdmissionRequest.Operation != admissionv1.Update {
		return admission.Allowed("Mutating only on create or update")
	}

	if r.isMutatoResource(&req) {
		return admission.Allowed("Not mutating mutato resources")
	}

	requestResponse := unknownResponse
	defer func() {
		r.logger.V(6).Info("mutation request processed", "response", requestResponse, "duration", time.Since(timeStart))
	}()

	// namespace is excluded from webhook using config
	isExcludedNamespace, err := r.skipExcludedNamespace(&req.AdmissionRequest)
	if err != nil {
		r.logger.Error(err, "error while excluding namespace")
	}

	if isExcludedNamespace {
		requestResponse = skipResponse
		return admission.Allowed("Namespace is set to be ignored by Gatekeeper config")
	}

	resp := r.mutateRequest(ctx, &req)
	requestResponse = successResponse
	return resp
}

func (r *Webhook) skipExcludedNamespace(req *admissionv1.AdmissionRequest) (bool, error) {
	obj := &unstructured.Unstructured{}
	if _, _, err := r.decoder.Decode(req.Object.Raw, nil, obj); err != nil {
		return false, err
	}
	obj.SetNamespace(req.Namespace)
	return false, nil
}

// isGatekeeperResource returns true if the request relates to a gatekeeper resource.
func (r *Webhook) isMutatoResource(req *admission.Request) bool {
	return req.AdmissionRequest.Kind.Group == "mutations.mutato.kubesphere.io"
}

// SetupWebhookWithManager sets up the webhook with the manager.
func (r *Webhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	r.logger = mgr.GetLogger().WithName("mutato-admission-webhook")
	r.client = mgr.GetClient()
	r.reader = mgr.GetAPIReader()
	r.decoder = serializer.NewCodecFactory(mgr.GetScheme()).UniversalDeserializer()
	mgr.GetWebhookServer().Register("/mutate", &webhook.Admission{Handler: r})
	return nil
}
