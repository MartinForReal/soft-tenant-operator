# soft-tenant-operator

## objectives

Cluster operator can develop and deploy authorization policy and secret content
Third party application can provision namespace for end user and develop authorization policy of namespace-scoped resources

## dependencies

github.com/fluxcd/helm-operator 

## models

### Tenant third party application

Tenant is a cluster-scoped resource which is defined by cluster-operator

When a tenant is provisioned,Following resources will be created.

* a management namespace

and following resources in management namespace

* a configMap which contains tenant metadata
* a role which is granted to crd resources in group of multitenant.mstech.com.cn and helm.fluxcd.io
* (TODO) a service account 
* (TODO) a roleBinding which refers to service account and role defined in the last step
* (TODO) a helm operator which have admin access to all of namespaces referred by templatenamespace resources in management namespace 

### TenantNamespace namespace reference which tenant has access to

TenantNamespace is a namespace reference object. One namespace can only be mapped to one tenantnamespace only
due to the limit of ownerReference in Kubernetes. (owner of the resource should reside in the same namespace or should be cluster-scoped) we cannot adopt ownerReference model provided by k8s api.

### Namespacetemplate policy crd which defines auto-provisioned resources for the target namespace

Namespacetemplate is a template resource which provide an anti-corruption layer for the underling templating technologies
By default it will generate a helm release crd which deploys resources in target namespace

### ClusterNamespaceTemplate crd which defines default namespacetemplate for each tenant

ClusterNamespaceTemplate is a template resource which defines a global policy 
