package ibmsecurityverify

import (
	"io"
	"bytes"
	"net/http"
    "testing"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ibmsecurityverify"
	corev1 "k8s.io/api/core/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/assert"
)

const (
	groupId = "testGroup"
	groupDisplayName = "testDisplayName"
	userId = "testUserId"
	userExternalId = "testExternalId"
	userName = "testUserName"
)

type HttpClientMock struct {
	mock.Mock
}

func (client *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := client.Called()
    return args.Get(0).(*http.Response), args.Error(1)
}

func TestGetGroupSuccess(t *testing.T) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecret.Data = make(map[string][]byte)
	credentialsSecret.Data["clientId"] = []byte("testClientId")
	credentialsSecret.Data["clientSecret"] = []byte("testClientSecret")

	httpClient := new(HttpClientMock)
	jsonResponse := `{ "accessToken": "token", "grantId": "grantId", "tokenType": "type", "expiresIn": 10000 }`
	mockAccessTokenResponse := &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(bytes.NewReader([]byte(jsonResponse))),
	}	
    httpClient.On("Do").Return(mockAccessTokenResponse, nil).Once()

	jsonResponse = `{ "id": "testGroup", "displayName": "testDisplayName", "members": [{ "id": "testUserId", "externalId": "testExternalId", "userName": "testUserName" }] }`
	mockGroupResponse := &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(bytes.NewReader([]byte(jsonResponse))),
	}	
    httpClient.On("Do").Return(mockGroupResponse, nil).Once()
	
	client := ibmsecurityverify.NewApiClient(credentialsSecret, httpClient)
    group := client.GetGroup("https://test.ibm.com", "testGroup")
	if assert.NotNil(t, group) {
		assert.Equal(t, groupId, group.Id)
		assert.Equal(t, groupDisplayName, group.DisplayName)
	}
}

func TestGetGroupFailureOnFetchingAccessToken(t *testing.T) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecret.Data = make(map[string][]byte)
	credentialsSecret.Data["clientId"] = []byte("testClientId")
	credentialsSecret.Data["clientSecret"] = []byte("testClientSecret")

	httpClient := new(HttpClientMock)
	jsonResponse := `{ "accessToken": "token", "grantId": "grantId", "tokenType": "type", "expiresIn": 10000 }`
	mockAccessTokenResponse := &http.Response{
		StatusCode: 400,
		Body: io.NopCloser(bytes.NewReader([]byte(jsonResponse))),
	}	
    httpClient.On("Do").Return(mockAccessTokenResponse, nil).Once()

	client := ibmsecurityverify.NewApiClient(credentialsSecret, httpClient)
    group := client.GetGroup("https://test.ibm.com", "testGroup")
	if assert.NotNil(t, group) {
		assert.Equal(t, "", group.Id)
		assert.Equal(t, "", group.DisplayName)
	}
}

func TestGetGroupFailureOnFetchingGroup(t *testing.T) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecret.Data = make(map[string][]byte)
	credentialsSecret.Data["clientId"] = []byte("testClientId")
	credentialsSecret.Data["clientSecret"] = []byte("testClientSecret")

	httpClient := new(HttpClientMock)
	jsonResponse := `{ "accessToken": "token", "grantId": "grantId", "tokenType": "type", "expiresIn": 10000 }`
	mockAccessTokenResponse := &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(bytes.NewReader([]byte(jsonResponse))),
	}	
    httpClient.On("Do").Return(mockAccessTokenResponse, nil).Once()

	jsonResponse = `{ "error": "test" }`
	mockGroupResponse := &http.Response{
		StatusCode: 400,
		Body: io.NopCloser(bytes.NewReader([]byte(jsonResponse))),
	}	
    httpClient.On("Do").Return(mockGroupResponse, nil).Once()
	
	client := ibmsecurityverify.NewApiClient(credentialsSecret, httpClient)
    group := client.GetGroup("https://test.ibm.com", "testGroup")
	if assert.NotNil(t, group) {
		assert.Equal(t, "", group.Id)
		assert.Equal(t, "", group.DisplayName)
	}
}
