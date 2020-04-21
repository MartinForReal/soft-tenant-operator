module github.com/minsheng-fintech-corp-ltd/soft-tenant-operator

go 1.13

require (
	github.com/fluxcd/helm-operator v1.0.1
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace github.com/docker/distribution => github.com/2opremio/distribution v0.0.0-20190419185413-6c9727e5e5de

replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce

replace github.com/fluxcd/helm-operator/pkg/install => github.com/fluxcd/helm-operator/pkg/install v0.0.0-20200415170445-493cde9b2223

replace k8s.io/client-go => k8s.io/client-go v0.17.2
