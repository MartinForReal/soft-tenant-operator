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

package multitenant

import (
	"context"
	"github.com/go-logr/logr"
	multitenantv1alpha1 "github.com/minsheng-fintech-corp-ltd/soft-tenant-operator/apis/multitenant/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceTemplateReconciler reconciles a Tenant object
type NamespaceTemplateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=namespacetemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=tenantnamespaces,verbs=get;list;watch;create;update;patch;delete

func (r *NamespaceTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("namespaceTemplate", req.NamespacedName)
	logger.Info("reconciling NamespaceTemplate")

	namespaceTemplate := &multitenantv1alpha1.NamespaceTemplate{}
	if err := r.Client.Get(ctx, req.NamespacedName, namespaceTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "NamespaceTemplate is not found when reconcile event")
		return ctrl.Result{}, err
	}

	// if label is not specified, don't trigger update
	if len(namespaceTemplate.Labels) <= 0 {
		logger.Info("skip trigger update because labels are empty")
		return ctrl.Result{}, nil
	}
	logger.Info("notify interested third party")
	tenantNamespaceList := &multitenantv1alpha1.TenantNamespaceList{}
	err := r.Client.List(ctx, tenantNamespaceList, client.MatchingLabelsSelector{Selector: labels.Everything()}, client.InNamespace(namespaceTemplate.Namespace))
	if err != nil {
		logger.Error(err, "failed to list tenantNamespacenamespace")
		return ctrl.Result{}, err
	}
	for _, item := range tenantNamespaceList.Items {
		if item.Spec.TemplateSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(item.Spec.TemplateSelector)
			if err != nil {
				logger.Error(err, "failed to parse selector")
				continue
			}
			if selector.Matches(labels.Set(namespaceTemplate.Labels)) {
				item.Labels = mergeMap(item.Labels, map[string]string{multitenantv1alpha1.NamespaceTemplateManagedObjectLabel + "-" + namespaceTemplate.Name: "initializing"})
				logger.Info("trigger an update for tenantNamespace", "namespace", item.Namespace, "name", item.Name)
				err = r.Client.Update(ctx, &item)
				if err != nil {
					logger.Error(err, "failed to reconcile error")
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *NamespaceTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenantv1alpha1.NamespaceTemplate{
			TypeMeta: ctrl.TypeMeta{
				Kind:       multitenantv1alpha1.NamespaceTemplateType,
				APIVersion: multitenantv1alpha1.GroupVersion.String(),
			},
		}).Complete(r)
}
