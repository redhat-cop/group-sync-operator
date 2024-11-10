package ibmsecurityverify

import (
	"io"
	"bytes"
	"net/http"
    "testing"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ibmsecurityverify"
	corev1 "k8s.io/api/core/v1"
	"github.com/stretchr/testify/mock"
)

type HttpClientMock struct {
	mock.Mock
}

func (client *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := client.Called()
    return args.Get(0).(*http.Response), args.Error(1)
}

func TestGetGroupMembersSuccess(t *testing.T) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecret.Data = make(map[string][]byte)
	credentialsSecret.Data["clientId"] = []byte("testClientId")
	credentialsSecret.Data["clientSecret"] = []byte("testClientSecret")

	jsonResponse := "{ \"accessToken\": \"token\",  \"grantId\": \"grantId\",  \"tokenType\": \"type\", \"expiresIn\": 10000 }"
	httpClient := new(HttpClientMock)
	mockResponse := &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(bytes.NewReader([]byte(jsonResponse))),
	}	
    httpClient.On("Do").Return(mockResponse, nil)

	client := ibmsecurityverify.NewApiClient(credentialsSecret, httpClient)
    client.GetGroupMembers("https://test.ibm.com", "testGroup")
}
