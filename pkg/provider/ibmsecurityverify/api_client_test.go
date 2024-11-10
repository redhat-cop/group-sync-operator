package ibmsecurityverify

import (
    "testing"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ibmsecurityverify"
	corev1 "k8s.io/api/core/v1"
)

func TestGetGroupMembersSuccess(t *testing.T) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 0
	httpClient := retryClient.StandardClient()

	credentialsSecret := &corev1.Secret{}
	credentialsSecret.Data = make(map[string][]byte)
	credentialsSecret.Data["clientId"] = []byte("testClientId")
	credentialsSecret.Data["clientSecret"] = []byte("testClientSecret")

	client := ibmsecurityverify.NewApiClient(credentialsSecret, httpClient)
    client.GetGroupMembers("https://test.ibm.com", "testGroup")
}
