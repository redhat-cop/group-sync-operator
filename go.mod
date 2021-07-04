module github.com/redhat-cop/group-sync-operator

go 1.16

require (
	github.com/Nerzal/gocloak/v5 v5.1.0
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec v0.19.3
	github.com/google/go-github/v32 v32.1.0
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/okta/okta-sdk-golang/v2 v2.3.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/library-go v0.0.0-20200527213645-a9b77f5402e3
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.10.0
	github.com/redhat-cop/operator-utils v1.1.4
	github.com/robfig/cron v0.0.0-20170526150127-736158dc09e1
	github.com/robfig/cron/v3 v3.0.1
	github.com/xanzy/go-gitlab v0.38.2
	github.com/yaegashi/msgraph.go v0.1.4
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/ldap.v2 v2.5.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	sigs.k8s.io/controller-runtime v0.8.3
)
