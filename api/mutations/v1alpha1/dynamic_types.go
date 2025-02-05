/*

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
	"github.com/open-policy-agent/gatekeeper/v3/pkg/mutation/match"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DynamicSpec struct {
	// Match allows the user to limit which resources get mutated.
	// Individual match criteria are AND-ed together. An undefined
	// match criteria matches everything.
	Match match.Match `json:"match,omitempty"`

	Rego string `json:"rego"`
}

type DynamicStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path="dynamic"
// +kubebuilder:resource:scope="Cluster"
// +kubebuilder:subresource:status

type Dynamic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicSpec   `json:"spec,omitempty"`
	Status DynamicStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DynamicList contains a list of Dynamic.
type DynamicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dynamic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dynamic{}, &DynamicList{})
}
