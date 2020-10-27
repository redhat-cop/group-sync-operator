module github.com/redhat-cop/group-sync-operator

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v36.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.3
	github.com/Azure/go-autorest/autorest/adal v0.8.1
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Nerzal/gocloak/v5 v5.1.0
	github.com/go-logr/logr v0.1.0
	github.com/google/go-github/v31 v31.0.0
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/okta/okta-sdk-golang/v2 v2.1.0
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/library-go v0.0.0-20200527213645-a9b77f5402e3
	github.com/operator-framework/operator-sdk v0.18.1
	github.com/redhat-cop/operator-utils v0.3.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/pflag v1.0.5
	github.com/xanzy/go-gitlab v0.30.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/ldap.v2 v2.5.1
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	sigs.k8s.io/controller-runtime v0.6.0

)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/xanzy/go-gitlab => github.com/xanzy/go-gitlab v0.31.0
	k8s.io/api => k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apiserver => k8s.io/apiserver v0.18.2
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator

)
