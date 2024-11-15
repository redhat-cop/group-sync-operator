package ibmsecurityverify

type IsvGroup struct {
	Id          string           `json:"id,omitempty"`
	DisplayName string           `json:"displayName,omitempty"`
	Members     []IsvGroupMember `json:"members,omitempty"`
}

type IsvGroupMember struct {
	Id         string `json:"id,omitempty"`
	ExternalId string `json:"externalId,omitempty"`
	UserName   string `json:"userName,omitempty"`
}
