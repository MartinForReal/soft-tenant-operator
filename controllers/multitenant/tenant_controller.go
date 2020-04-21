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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	multitenantv1alpha1 "github.com/minsheng-fintech-corp-ltd/soft-tenant-operator/apis/multitenant/v1alpha1"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=tenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multitenant.mstech.com.cn,resources=tenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *TenantReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("tenant", req.NamespacedName)
	logger.Info("reconciling tenant")

	tenant := &multitenantv1alpha1.Tenant{}
	if err := r.Client.Get(ctx, req.NamespacedName, tenant); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "tenant is not found when reconcile event")
		return ctrl.Result{}, err
	}
	if !tenant.DeletionTimestamp.IsZero() {
		//wait for deletion
		logger.Info("tenant is being deleted")
		return ctrl.Result{}, nil
	}
	existingObj := tenant.DeepCopy()

	// ensure namespace object
	var namespace *corev1.Namespace
	// search for existing namespace
	if len(tenant.Status.Name) <= 0 {
		logger.Info("status is empty, search for existing managed namespace")
		namespaceList := &corev1.NamespaceList{}
		err := r.Client.List(ctx, namespaceList, client.MatchingLabels(tenant.GetManagedLabels()))
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
					GenerateName: tenant.Name + "-",
					Labels:       tenant.GetManagedLabels(),
				},
			}
		}
	} else {
		logger.Info("current status is found")
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   tenant.Status.Name,
				Labels: tenant.GetManagedLabels(),
			},
		}
	}

	// if namespace is provisioned, use existing one.
	if len(tenant.Status.Name) <= 0 {
		logger.Info("namespace is not created,create one")
		if tenant.Spec.IsNotGeneratedName {
			namespace.Name = tenant.Name
		} else {
			namespace.GenerateName = tenant.Name + "-"
		}
	}
	namespaceOpsResult, err := ensureObject(ctx, r.Client, r.Scheme, namespace, tenant, true, logger)
	if err != nil {
		logger.Error(err, "failed to ensure namespace")
		return ctrl.Result{}, err
	}
	if !namespace.DeletionTimestamp.IsZero() {
		logger.Info("namespace is being deleted,ignore", "namespace", namespace)
		return ctrl.Result{}, nil
	}
	tenant.Status.Name = namespace.Name
	// ensure service account
	serviceAccout := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: namespace.Name,
		},
	}
	_, err = ensureObject(ctx, r.Client, r.Scheme, serviceAccout, tenant, true, logger)
	if err != nil {
		logger.Error(err, "failed to ensure serviceaccount")
		return ctrl.Result{}, err
	}

	rolebinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: namespace.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      tenant.Name,
				Namespace: namespace.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "admin",
		},
	}
	_, err = ensureObject(ctx, r.Client, r.Scheme, rolebinding, tenant, true, logger)
	if err != nil {
		logger.Error(err, "failed to ensure rolebinding")
		return ctrl.Result{}, err
	}

	//add metadata
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name + "-config",
			Namespace: namespace.Name,
		},
		Data: map[string]string{
			multitenantv1alpha1.TenantMetadataTenantNameKey: tenant.Name,
		},
	}
	_, err = ensureObject(ctx, r.Client, r.Scheme, configMap, tenant, true, logger)
	if err != nil {
		logger.Error(err, "failed to ensure configmap")
		return ctrl.Result{}, err
	}
	// update status
	if !equality.Semantic.DeepEqual(existingObj, tenant) {
		tenant.Status.Name = namespace.Name
		err = r.Client.Status().Update(ctx, tenant)
		if err != nil {
			if namespaceOpsResult == controllerutil.OperationResultCreated {
				r.Client.Delete(ctx, namespace)
			}
		}
	}

	return ctrl.Result{}, err
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenantv1alpha1.Tenant{
			TypeMeta: ctrl.TypeMeta{
				Kind:       multitenantv1alpha1.TenantType,
				APIVersion: multitenantv1alpha1.GroupVersion.String(),
			},
		}).Owns(&corev1.Namespace{
		TypeMeta: ctrl.TypeMeta{
			Kind:       "Namespace",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
	}).Owns(&corev1.ServiceAccount{
		TypeMeta: ctrl.TypeMeta{
			Kind:       rbacv1.ServiceAccountKind,
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
	}).Owns(&rbacv1.RoleBinding{
		TypeMeta: ctrl.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
	}).Owns(&corev1.ConfigMap{
		TypeMeta: ctrl.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.GroupName,
		},
	}).Complete(r)
}
