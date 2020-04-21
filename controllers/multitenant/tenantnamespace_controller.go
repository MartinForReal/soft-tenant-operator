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
	"errors"
	helmReleasev1 "github.com/fluxcd/helm-operator/pkg/apis/helm.fluxcd.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	multitenantv1alpha1 "github.com/minsheng-fintech-corp-ltd/soft-tenant-operator/apis/multitenant/v1alpha1"
)

// TenantNamespaceReconciler reconciles a TenantNamespace object
type TenantNamespaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const DefaultWaitTimeout = 15 * time.Second

// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=tenantnamespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=clusternamespacetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=namespacetemplates,verbs=get;list;watch

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=helm.fluxcd.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete

func (r *TenantNamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("tenantnamespace", req.NamespacedName)
	logger.Info("reconciling tenantnamespace")
	tenantNs := &multitenantv1alpha1.TenantNamespace{}
	if err := r.Client.Get(ctx, req.NamespacedName, tenantNs); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get tenantnamespace when reconcile event")
		return ctrl.Result{}, err
	}
	if !tenantNs.DeletionTimestamp.IsZero() {
		//wait for deletion
		if len(tenantNs.Status.Name) > 0 {
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: tenantNs.Status.Name,
				},
			}
			if err := r.Client.Delete(ctx, namespace); err != nil {
				logger.Error(err, "failed to delete ns")
				//ignore error here
			}
		}
		controllerutil.RemoveFinalizer(tenantNs, multitenantv1alpha1.TenantNamespaceFinalizer)
		return ctrl.Result{}, r.Client.Update(ctx, tenantNs)
	}
	// add finalizer
	existobj := tenantNs.DeepCopyObject()
	controllerutil.AddFinalizer(tenantNs, multitenantv1alpha1.TenantNamespaceFinalizer)
	if !equality.Semantic.DeepEqual(existobj, tenantNs) {
		// add finalizer
		logger.Info("finalizer is added to object")
		return ctrl.Result{}, r.Client.Update(ctx, tenantNs)
	}

	// ensure template is managed by tenant
	ownerNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: tenantNs.Namespace,
		},
	}
	key, _ := client.ObjectKeyFromObject(ownerNamespace)
	if err := r.Client.Get(ctx, key, ownerNamespace); err != nil {
		logger.Error(err, "failed to get owner namespace")
		return ctrl.Result{}, err
	}
	if !ownerNamespace.DeletionTimestamp.IsZero() {
		logger.Info("namespace is being deleted,ignored")
		return ctrl.Result{}, nil
	}

	found := false
	expectedVersion := multitenantv1alpha1.GroupVersion.String()
	for _, item := range ownerNamespace.OwnerReferences {
		if item.APIVersion == expectedVersion && item.Kind == multitenantv1alpha1.TenantType {
			found = true
			break
		}
	}
	if !found {
		logger.Info("found tenantnamespace outside of management ns,deleted", "tenantns", tenantNs)
		return ctrl.Result{}, r.Client.Delete(ctx, tenantNs)
	}

	// ensure namespace object
	var namespace *corev1.Namespace
	// search for existing namespace
	if len(tenantNs.Status.Name) <= 0 {
		logger.Info("status is empty, search for existing managed namespace")
		namespaceList := &corev1.NamespaceList{}
		err := r.Client.List(ctx, namespaceList, client.MatchingLabels(tenantNs.GetManagedLabels()))
		if err != nil {
			logger.Error(err, "failed to list namespace")
			return ctrl.Result{}, err
		}
		if len(namespaceList.Items) > 1 {
			err := errors.New("multiple namespace instance found ")
			logger.Error(err, "manually delete namespace if necessary")
			return ctrl.Result{}, err
		} else if len(namespaceList.Items) == 1 {
			logger.Info("found existing namespace", "namespace", namespaceList.Items[0].Name)
			namespace = &namespaceList.Items[0]
		} else {
			logger.Info("didn't found existing namespace")
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: tenantNs.Name + "-",
					Labels:       tenantNs.GetManagedLabels(),
				},
			}
		}
	} else {
		logger.Info("current status is found")
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tenantNs.Status.Name,
			},
		}
	}

	if len(namespace.Name) <= 0 {
		logger.Info("namespace is not created,create one")
		if tenantNs.Spec.IsNotGeneratedName {
			namespace.ObjectMeta.Name = tenantNs.Name
		}
	}

	// ensure namespace
	// don't create owner reference due to k8s limitation: owner should resides in the same namespace
	namespaceOpsResult, err := ensureObject(ctx, r.Client, r.Scheme, namespace, tenantNs, false, logger)
	if err != nil {
		logger.Error(err, "failed to ensure namespace")
		return ctrl.Result{}, err
	}
	if !namespace.DeletionTimestamp.IsZero() {
		logger.Info("namespace is being deleted,ignore", "namespace", namespace)
		return ctrl.Result{}, nil
	}
	tenantNs.Status.Name = namespace.Name

	//list existing helm release
	helmList := &helmReleasev1.HelmReleaseList{}
	err = r.Client.List(ctx, helmList, client.MatchingLabels(tenantNs.GetManagedLabels()), client.InNamespace(tenantNs.Namespace))
	if err != nil {
		logger.Error(err, "failed to get existing helm release")
		return ctrl.Result{}, nil
	}
	existingHelmReleaseList := make(map[string]*helmReleasev1.HelmRelease)
	for _, item := range helmList.Items {
		obj := item
		existingHelmReleaseList[item.Name] = &obj
	}
	//apply cluster template
	if tenantNs.Spec.ClusterTemplateSelector != nil {
		logger.Info("start to applying clusternamespacetemplate")
		clusterNamespaceTemplateList := &multitenantv1alpha1.ClusterNamespaceTemplateList{}
		clusterNsTemplateSelector, err := metav1.LabelSelectorAsSelector(tenantNs.Spec.ClusterTemplateSelector)
		if err != nil {
			logger.Error(err, "invalid cluster namespace template")
		} else if err := r.Client.List(ctx, clusterNamespaceTemplateList, client.MatchingLabelsSelector{Selector: clusterNsTemplateSelector}); err != nil {
			logger.Error(err, "failed to list cluster namespace using selector", "selector", clusterNsTemplateSelector)
		} else {
			for _, item := range clusterNamespaceTemplateList.Items {
				logger.Info("apply clusterNamespaceTemplate", "clusterNamespaceTemplate", item.Name)
				selector := labels.SelectorFromSet(labels.Set(mergeMap(tenantNs.GetManagedLabels(), item.GetManagedLabels())))
				values, err := getConfigReference(ctx, r.Client, logger, item.Namespace, selector)
				if err != nil {
					logger.Error(err, "failed to get configuration")
					continue
				}
				helmRelease := getHelmRelease(tenantNs.Name+"-cnt-"+item.Name, "cnt-"+item.Name, tenantNs.Namespace, namespace.Name, item.Spec.TemplateChartSource, values, item.GetManagedLabels())
				_, err = ensureObject(ctx, r.Client, r.Scheme, helmRelease, tenantNs, true, logger)
				if err != nil {
					logger.Error(err, "failed to ensure helm release", "release", helmRelease)
					continue
				}
				delete(tenantNs.Labels, multitenantv1alpha1.ClusterNamespaceTemplateManagedObjectLabel+"-"+item.Name)
				logger.Info("created helm release", "clustertemplate", item.Name, "release", helmRelease.Name)
				delete(existingHelmReleaseList, helmRelease.Name)
			}
		}
	}
	// apply namespace template
	if tenantNs.Spec.TemplateSelector != nil {
		logger.Info("start to applying namespacetemplate")
		namespaceTemplateList := &multitenantv1alpha1.NamespaceTemplateList{}
		namespaceTemplateSelector, err := metav1.LabelSelectorAsSelector(tenantNs.Spec.TemplateSelector)
		if err != nil {
			logger.Error(err, "invalid namespace template")
		} else if err := r.Client.List(ctx, namespaceTemplateList, client.MatchingLabelsSelector{Selector: namespaceTemplateSelector}, client.InNamespace(tenantNs.Namespace)); err != nil {
			logger.Error(err, "failed to list cluster namespace using selector", "selector", namespaceTemplateSelector)
		} else {
			for _, item := range namespaceTemplateList.Items {
				logger.Info("apply namespaceTemplate", "namespaceTemplate", item.Name)

				selector := labels.SelectorFromSet(labels.Set(mergeMap(tenantNs.GetManagedLabels(), item.GetManagedLabels())))
				values, err := getConfigReference(ctx, r.Client, logger, item.Namespace, selector)
				if err != nil {
					logger.Error(err, "failed to get configuration")
					continue
				}
				// get configMap and secret for template
				helmRelease := getHelmRelease(tenantNs.Name+"-nt-"+item.Name, item.Name, tenantNs.Namespace, namespace.Name, item.Spec.TemplateChartSource, values, mergeMap(item.GetManagedLabels()))
				_, err = ensureObject(ctx, r.Client, r.Scheme, helmRelease, tenantNs, true, logger)
				if err != nil {
					logger.Error(err, "failed to ensure helm release", "release", helmRelease)
					continue
				}
				delete(tenantNs.Labels, multitenantv1alpha1.NamespaceTemplateManagedObjectLabel+"-"+item.Name)
				logger.Info("created helm release", "template", item.Name, "release", helmRelease.Name)
				delete(existingHelmReleaseList, helmRelease.Name)
			}
		}
	}

	for key, item := range existingHelmReleaseList {
		logger.Info("delete helm release because selector is not applied", "release", key)
		r.Client.Delete(ctx, item)
	}

	tenantNs.Labels = make(map[string]string)
	if !equality.Semantic.DeepEqual(existobj, tenantNs) {
		err := r.Client.Update(ctx, tenantNs)
		if err != nil {
			if namespaceOpsResult == controllerutil.OperationResultCreated {
				r.Client.Delete(ctx, namespace)
			}
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TenantNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenantv1alpha1.TenantNamespace{
			TypeMeta: ctrl.TypeMeta{
				Kind:       multitenantv1alpha1.TenantNamespaceType,
				APIVersion: multitenantv1alpha1.GroupVersion.String(),
			},
		}).
		Watches(&source.Kind{Type: &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(mapNamespaceToTenantNamespace)}).
		Owns(&helmReleasev1.HelmRelease{
			TypeMeta: metav1.TypeMeta{
				APIVersion: helmReleasev1.SchemeGroupVersion.String(),
				Kind:       "HelmRelease",
			},
		}).
		Complete(r)
}

func mapNamespaceToTenantNamespace(a handler.MapObject) []reconcile.Request {
	labels := a.Meta.GetLabels()
	if name, ok := labels[multitenantv1alpha1.TenantNamespaceManagedNSLabelName]; ok {
		if namespace, ok := labels[multitenantv1alpha1.TenantNamespaceManagedNSLabelNamespace]; ok {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				},
			}
		}
	}
	return nil
}

func getHelmRelease(name, releaseName, namespace, targetNamespace string, chartSource helmReleasev1.ChartSource, valueFrom []helmReleasev1.ValuesFromSource, labels map[string]string) *helmReleasev1.HelmRelease {
	return &helmReleasev1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: helmReleasev1.HelmReleaseSpec{
			HelmVersion:     helmReleasev1.HelmV3,
			ChartSource:     chartSource,
			ValuesFrom:      valueFrom,
			ReleaseName:     releaseName,
			TargetNamespace: targetNamespace,
			SkipCRDs:        true,
			Wait:            true,
			ForceUpgrade:    true,
			Values:          helmReleasev1.HelmValues{Data: map[string]interface{}{}},
			Rollback: helmReleasev1.Rollback{
				Enable:       true,
				Retry:        true,
				Recreate:     true,
				DisableHooks: true,
			},
		},
	}
}

func getConfigReference(ctx context.Context, cli client.Client, logger logr.Logger, namespace string, selector labels.Selector) ([]helmReleasev1.ValuesFromSource, error) {
	var result []helmReleasev1.ValuesFromSource
	cmList := &corev1.ConfigMapList{}
	err := cli.List(ctx, cmList, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		logger.Error(err, "failed to list resource")
		return nil, err
	}
	for _, item := range cmList.Items {
		result = append(result, helmReleasev1.ValuesFromSource{
			ConfigMapKeyRef: &helmReleasev1.OptionalConfigMapKeySelector{
				ConfigMapKeySelector: helmReleasev1.ConfigMapKeySelector{
					Namespace: namespace,
					LocalObjectReference: helmReleasev1.LocalObjectReference{
						Name: item.Name,
					},
					Key: "values.yaml",
				},
				Optional: true,
			},
		})
	}
	secretList := &corev1.SecretList{}
	err = cli.List(ctx, cmList, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		logger.Error(err, "failed to list resource")
		return nil, err
	}
	for _, item := range secretList.Items {
		result = append(result, helmReleasev1.ValuesFromSource{
			SecretKeyRef: &helmReleasev1.OptionalSecretKeySelector{
				SecretKeySelector: helmReleasev1.SecretKeySelector{
					Namespace: namespace,
					LocalObjectReference: helmReleasev1.LocalObjectReference{
						Name: item.Name,
					},
					Key: "values.yaml",
				},
				Optional: true,
			},
		})
	}
	return result, nil
}
