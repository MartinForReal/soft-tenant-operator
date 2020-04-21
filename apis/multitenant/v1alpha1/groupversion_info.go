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

// Package v1alpha1 contains API Schema definitions for the multitenant v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=multitenant.mstech.com.cn
// +kubebuilder:validation:Required
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "multitenant.mstech.com.cn", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func GetMultitenantManagedLabels() map[string]string {
	return map[string]string{
		"multitenant.mstech.com.cn/managed": "true",
	}
}

// +kubebuilder:object:generate:=false
type K8sObject interface {
	metav1.Object
	runtime.Object
}

// +kubebuilder:object:generate:=false
type MultiTenancyObjectAsOwner interface {
	metav1.Object
	GetManagedLabels() map[string]string
}
