/*
Copyright 2020 Min Sheng Fintech Corp.ltd.

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
	helmReleasev1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const NamespaceTemplateType = "NamespaceTemplate"
const NamespaceTemplateManagedObjectLabel = "multitenant/nt"

// NamespaceTemplateSpec defines the desired state of NamespaceTemplate
type NamespaceTemplateSpec struct {
	TemplateChartSource helmReleasev1.ChartSource `json:"templateChartSource"`
}

// NamespaceTemplateStatus defines the observed state of NamespaceTemplate
type NamespaceTemplateStatus struct {
}

type NamespaceTemplateRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Tenant    string `json:"tenant"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=CreationTimestamp,type=date
// NamespaceTemplate is the Schema for the namespacetemplates API
type NamespaceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceTemplateSpec   `json:"spec,omitempty"`
	Status NamespaceTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NamespaceTemplateList contains a list of NamespaceTemplate
type NamespaceTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NamespaceTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NamespaceTemplate{}, &NamespaceTemplateList{})
}
func (r *NamespaceTemplate) GetManagedLabels() map[string]string {
	result := GetMultitenantManagedLabels()
	result[NamespaceTemplateManagedObjectLabel+"-"+r.GetName()] = r.GetName()
	return result
}
