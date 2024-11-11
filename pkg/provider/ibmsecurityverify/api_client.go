package ibmsecurityverify

import (
	"fmt"
	"encoding/json"
	"strings"
	"net/url"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	corev1 "k8s.io/api/core/v1"
)

var (
	logger = logf.Log.WithName("ibm_security_verify_api_client")
)

type IbmSecurityVerifyClient interface {
	SetHttpClient(client HttpClient)
	SetCredentialsSecret(secret *corev1.Secret)
	GetGroup(tenantUrl string, groupId string) IsvGroup
}

type HttpClient interface {
    Do(req *http.Request) (*http.Response, error)
}

type ApiClient struct {
	credentialsSecret *corev1.Secret
	httpClient HttpClient
}

type accessTokenResponse struct {
	AccessToken string
	GrantId string
	TokenType string
	ExpiresIn int 
}

func (apiClient *ApiClient) SetHttpClient(client HttpClient) {
	apiClient.httpClient = client
}

func (apiClient *ApiClient) SetCredentialsSecret(secret *corev1.Secret) {
	apiClient.credentialsSecret = secret
}

func (apiClient *ApiClient) GetGroup(tenantUrl string, groupId string) IsvGroup {
	token := apiClient.fetchAccessToken(tenantUrl)
	var group IsvGroup
	if token != "" {
		group = apiClient.fetchGroup(token, tenantUrl, groupId)
	}
	return group
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
	var accessToken string
	if err != nil || response.StatusCode != 200 {
		logger.Error(err, fmt.Sprint("Failed to request API access token. Response code: %d", response.StatusCode))
	} else {
		decoder := json.NewDecoder(response.Body)
		var data accessTokenResponse
		err = decoder.Decode(&data)
		if err == nil {
			accessToken = data.AccessToken
		} else {
			logger.Error(err, fmt.Sprint("Failed to decode access token response"))
		}
	}
	defer response.Body.Close()
	return accessToken
}

func (apiClient *ApiClient) fetchGroup(accessToken string, tenantUrl string, groupId string) IsvGroup {
	groupUrl := fmt.Sprintf("%s/v2.0/Groups/%s?membershipType=firstLevelUsersAndGroups", tenantUrl, groupId)
	logger.Info(fmt.Sprintf("Requesting members from group %s from %s", groupId, groupUrl))
	request, err := http.NewRequest("GET", groupUrl, nil)
	request.Header.Add("accept", "application/scim+json")
	request.Header.Add("authorization", "bearer " + accessToken)
	response, err := apiClient.httpClient.Do(request)
	var group IsvGroup
	if err != nil || response.StatusCode != 200 {
		logger.Error(err, fmt.Sprint("Failed to fetch group %s. Response code: %d", groupId, response.StatusCode))
	} else {
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&group)
		if err != nil {
			logger.Error(err, fmt.Sprint("Failed to decode group response"))
		}
	}
	defer response.Body.Close()
	return group
}
