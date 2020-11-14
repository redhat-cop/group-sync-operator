module github.com/redhat-cop/group-sync-operator

go 1.13

require (
	github.com/Nerzal/gocloak/v5 v5.1.0
	github.com/go-logr/logr v0.1.0
	github.com/google/go-github/v32 v32.1.0
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/library-go v0.0.0-20200527213645-a9b77f5402e3
	github.com/operator-framework/operator-lib v0.1.0
	github.com/prometheus/common v0.9.1
	github.com/redhat-cop/operator-utils v0.3.6
	github.com/robfig/cron v0.0.0-20170526150127-736158dc09e1
	github.com/robfig/cron/v3 v3.0.1
	github.com/xanzy/go-gitlab v0.38.2
	github.com/yaegashi/msgraph.go v0.1.4
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/ldap.v2 v2.5.1
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	k8s.io/api => k8s.io/api v0.18.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.6
	k8s.io/client-go => k8s.io/client-go v0.18.6

)
