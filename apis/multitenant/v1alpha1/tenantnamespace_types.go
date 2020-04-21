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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const TenantNamespaceType = "TenantNamespace"
const TenantNamespaceFinalizer = "tenantnamespace.multitenant.mstech.com.cn"
const TenantNamespaceManagedNSLabelName = "tenantnamespace.multitenant.mstech.com.cn/name"
const TenantNamespaceManagedNSLabelNamespace = "tenantnamespace.multitenant.mstech.com.cn/namespace"

// TenantNamespaceSpec defines the desired state of TenantNamespace
type TenantNamespaceSpec struct {
	// +optional
	IsNotGeneratedName bool `json:"isNotGeneratedName,omitempty"`

	// +optional
	ClusterTemplateSelector *metav1.LabelSelector `json:"clusterTemplateSelector,omitempty"`

	// +optional
	TemplateSelector *metav1.LabelSelector `json:"templateSelector,omitempty"`
}

// TenantNamespaceStatus defines the observed state of TenantNamespace
type TenantNamespaceStatus struct {
	// +optional
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".status.name",name=Namespace,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=CreationTimestamp,type=date

// TenantNamespace is the Schema for the tenantnamespaces API
type TenantNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantNamespaceSpec   `json:"spec,omitempty"`
	Status TenantNamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantNamespaceList contains a list of TenantNamespace
type TenantNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantNamespace{}, &TenantNamespaceList{})
}

func (r *TenantNamespace) GetManagedLabels() map[string]string {
	result := GetMultitenantManagedLabels()
	result[TenantNamespaceManagedNSLabelName] = r.Name
	result[TenantNamespaceManagedNSLabelNamespace] = r.Namespace
	return result
}
