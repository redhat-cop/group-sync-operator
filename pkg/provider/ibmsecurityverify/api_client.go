package ibmsecurityverify

import (
	"fmt"
	"encoding/json"
	"strings"
	"net/url"
	"net/http"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	corev1 "k8s.io/api/core/v1"
)

var (
	logger = logf.Log.WithName("ibm_security_verify_api_client")
)

type IbmSecurityVerifyClient interface {
	GetGroupMembers(tenantUrl string, groupId string) []string
}

type ApiClient struct {
	credentialsSecret *corev1.Secret
	httpClient *http.Client
}

func NewApiClient(credentialsSecret *corev1.Secret, httpClient *http.Client) IbmSecurityVerifyClient {
    return &ApiClient{credentialsSecret, httpClient}
}

type AccessTokenResponse struct {
	AccessToken string
	GrantId string
	TokenType string
	ExpiresIn int 
}

func (apiClient *ApiClient) GetGroupMembers(tenantUrl string, groupId string) []string {
	apiClient.getAccessToken(tenantUrl)
	var array []string
    array[0] = "test"
	return array
}

func (apiClient *ApiClient) buildHttpClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	return retryClient.StandardClient()
}

func (apiClient *ApiClient) getAccessToken(tenantUrl string) string {
	tokenUrl := tenantUrl + "/v1.0/endpoint/default/token"
	logger.Info(fmt.Sprint("Requesting API access token from %s", tokenUrl))
	requestData := url.Values{}
    requestData.Set("client_id", string(apiClient.credentialsSecret.Data["clientId"]))
    requestData.Set("client_secret", string(apiClient.credentialsSecret.Data["clientSecret"]))
	request, error := http.NewRequest("POST", tokenUrl, strings.NewReader(requestData.Encode()))
	request.Header.Add("accept", "application/scim+json")
	response, error := apiClient.httpClient.Do(request)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
    var data AccessTokenResponse
    error = decoder.Decode(&data)
	if error != nil {
		// TODO
	}
	return data.AccessToken
}
