package ibmsecurityverify

type IsvGroup struct {
	Id string
	DisplayName string
	Members []IsvGroupMember
}

type IsvGroupMember struct {
	Id string
	ExternalId string
	UserName string
}
