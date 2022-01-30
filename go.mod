module github.com/redhat-cop/group-sync-operator

go 1.16

require (
	github.com/Nerzal/gocloak/v5 v5.5.0
	github.com/go-logr/logr v1.2.0
	github.com/go-openapi/spec v0.19.3
	github.com/google/go-github/v39 v39.2.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/okta/okta-sdk-golang/v2 v2.3.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.18.1
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/library-go v0.0.0-20200527213645-a9b77f5402e3
	github.com/palantir/go-githubapp v0.9.2-0.20210913152418-062be9630ea5
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/redhat-cop/operator-utils v1.1.4
	github.com/robfig/cron v0.0.0-20170526150127-736158dc09e1
	github.com/robfig/cron/v3 v3.0.1
	github.com/shurcooL/githubv4 v0.0.0-20210725200734-83ba7b4c9228
	github.com/xanzy/go-gitlab v0.54.3
	github.com/yaegashi/msgraph.go v0.1.4
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3
	gopkg.in/ldap.v2 v2.5.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.23.3
	k8s.io/client-go v0.20.2
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65
	sigs.k8s.io/controller-runtime v0.8.3
)
