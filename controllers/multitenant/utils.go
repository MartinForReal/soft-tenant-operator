package multitenant

import (
	"context"
	"github.com/go-logr/logr"
	multitenantv1alpha1 "github.com/minsheng-fintech-corp-ltd/soft-tenant-operator/apis/multitenant/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func mergeMap(existing map[string]string, requiredSet ...map[string]string) map[string]string {
	if existing == nil {
		existing = make(map[string]string)
	}
	for _, required := range requiredSet {
		for k, v := range required {
			if existingV, ok := existing[k]; !ok || v != existingV {
				existing[k] = v
			}
		}
	}
	return existing
}

func ensureObject(ctx context.Context, client client.Client, scheme *runtime.Scheme, obj multitenantv1alpha1.K8sObject, owner multitenantv1alpha1.MultiTenancyObjectAsOwner, SetOwnerReference bool, logger logr.Logger) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrUpdate(ctx, client, obj, func() (err error) {
		if SetOwnerReference {
			if err = controllerutil.SetControllerReference(owner, obj, scheme); err != nil {
				logger.Error(err, "failed to setControllerReference")
				return err
			}
		}
		obj.SetLabels(mergeMap(owner.GetLabels(), owner.GetManagedLabels(), obj.GetLabels()))
		obj.SetAnnotations(mergeMap(owner.GetAnnotations(), obj.GetAnnotations()))
		return nil
	})
}
