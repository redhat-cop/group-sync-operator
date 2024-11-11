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
	GetGroup(tenantUrl string, groupId string) IsvGroup
}

type HttpClient interface {
    Do(req *http.Request) (*http.Response, error)
}

type ApiClient struct {
	credentialsSecret *corev1.Secret
	httpClient HttpClient
}

func NewApiClient(credentialsSecret *corev1.Secret, httpClient HttpClient) IbmSecurityVerifyClient {
    return &ApiClient{credentialsSecret, httpClient}
}

type accessTokenResponse struct {
	accessToken string
	grantId string
	tokenType string
	expiresIn int 
}

func (apiClient *ApiClient) GetGroup(tenantUrl string, groupId string) IsvGroup {
	token := apiClient.fetchAccessToken(tenantUrl)
	return apiClient.fetchGroup(token, tenantUrl, groupId)
}

func (apiClient *ApiClient) buildHttpClient() HttpClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	return retryClient.StandardClient()
}

func (apiClient *ApiClient) fetchAccessToken(tenantUrl string) string {
	tokenUrl := tenantUrl + "/v1.0/endpoint/default/token"
	logger.Info(fmt.Sprintf("Requesting API access token from %s", tokenUrl))
	requestData := url.Values{}
    requestData.Set("client_id", string(apiClient.credentialsSecret.Data["clientId"]))
    requestData.Set("client_secret", string(apiClient.credentialsSecret.Data["clientSecret"]))
	request, _ := http.NewRequest("POST", tokenUrl, strings.NewReader(requestData.Encode()))
	request.Header.Add("accept", "application/scim+json")
	response, err := apiClient.httpClient.Do(request)
	responseCode := response.StatusCode
	if response.StatusCode != 200 {
		logger.Error(err, fmt.Sprint("Failed to request API access token. Response code: %d", responseCode))
	}
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(response.Body)
    var data accessTokenResponse
    err = decoder.Decode(&data)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	return data.accessToken
}

func (apiClient *ApiClient) fetchGroup(accessToken string, tenantUrl string, groupId string) IsvGroup { // TODO return type
	groupUrl := fmt.Sprintf("%s/v2.0/Groups/%s?membershipType=firstLevelUsersAndGroups", tenantUrl, groupId)
	logger.Info(fmt.Sprintf("Requesting members from group %s from %s", groupId, groupUrl))
	request, err := http.NewRequest("GET", groupUrl, nil)
	if err != nil {
		panic(err)
	}
	request.Header.Add("accept", "application/scim+json")
	request.Header.Add("authorization", "bearer " + accessToken)
	response, err := apiClient.httpClient.Do(request)
	decoder := json.NewDecoder(response.Body)
    var group IsvGroup
    err = decoder.Decode(&group)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	return group
}
