package model

import (
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
)

func (TableUser) TableName() string {
	return "users"
}

// swagger:model CreateUserStruct
type TableUser struct {
	Id int `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"-"`
	// required: true
	UserName string `gorm:"column:username;type:varchar(100);unique_index:idx_username;not null" json:"username" validate:"required"`
	// example: 10
	// required: true
	PartId int `gorm:"column:partid;type:int;default:10;not null" json:"partid"`
	// required: true
	Email string `gorm:"column:email;type:varchar(250);not null" json:"email" validate:"required,email"`
	// required: true
	Password string `gorm:"-" json:"password"`
	// required: true

	FirstName string `gorm:"column:firstname;type:varchar(50);not null" json:"firstname" validate:"required"`
	// required: true
	LastName string `gorm:"column:lastname;type:varchar(50);not null" json:"lastname"`
	// required: true
	// example: NOC
	Department string `gorm:"column:department;type:varchar(50);not null" json:"department" validate:"required"`
	// required: true
	// example: admin
	UserGroup    string `gorm:"column:usergroup;type:varchar(250);not null" json:"usergroup" validate:"required"`
	IsAdmin      bool   `gorm:"-" json:"-"`
	ExternalAuth bool   `gorm:"-" json:"-"`

	Hash string `gorm:"column:hash;type:varchar(128);not null" json:"-"`
	// should be a unique value representing user
	// example: e71771a2-1ea0-498f-8d27-391713e10664
	// required: true
	GUID            string    `gorm:"column:guid;type:varchar(50);not null" json:"guid" validate:"required"`
	CreatedAt       time.Time `gorm:"column:created_at;default:current_timestamp;not null" json:"-"`
	ExternalProfile string    `gorm:"-" json:"-"`
	Avatar          string    `gorm:"-" json:"-"`
}

type HTTPAUTHResp struct {
	Auth bool      `json:"auth" validate:"required"`
	Data TableUser `json:"data" validate:"required"`
}

// swagger:model UserLoginSuccessResponse
type UserTokenSuccessfulResponse struct {
	// the token
	// example: JWT Token
	Token string `json:"token"`
	// the uuid
	// example: b9f6q23a-0bde-41ce-cd36-da3dbc17ea12
	Scope string `json:"scope"`
	// the uuid
	User struct {
		Admin bool `json:"admin"`
	} `json:"user"`
}

// swagger:model FailureResponse
type UserTokenBadResponse struct {
	// statuscode
	StatusCode int `json:"statuscode"`
	// errot
	Error string `json:"error"`
	// message
	Message string `json:"message"`
}

// swagger:model ListUsers
type GetUser struct {
	// count
	Count int `json:"count"`
	// the data
	Data []*TableUser `json:"data"`
}

// swagger:model ListExternalUsers
type GetExternalUser struct {
	// count
	Count int `json:"count"`
	// the data
	Data []TableUser `json:"data"`
}

// swagger:model UserLogin
type UserloginDetails struct {
	// example: admin
	// required: true
	Username string `json:"username" validate:"required"`
	// example: sipcapture
	// required: true
	Password string `json:"password" validate:"required"`
}

// swagger:model UserCreateSuccessResponse
type UserCreateSuccessfulResponse struct {
	// example: af72057b-2745-0a1b-b674-56586aadec57
	Data string `json:"data"`
	// example: successfully created user
	Message string `json:"message"`
}

// swagger:model UserUpdateSuccessResponse
type UserUpdateSuccessfulResponse struct {
	// example: af72057b-2745-0a1b-b674-56586aadec57
	Data string `json:"data"`
	// example: successfully updated user
	Message string `json:"message"`
}

// swagger:model UserDeleteSuccessResponse
type UserDeleteSuccessfulResponse struct {
	// example: af72057b-2745-0a1b-b674-56586aadec57
	Data string `json:"data"`
	// example: successfully deleted user
	Message string `json:"message"`
}

// swagger:model OAuth2TokenExchange
type OAuth2TokenExchange struct {
	// example: token
	// required: true
	OneTimeToken string `json:"token" validate:"required"`
}

// swagger:model OAuth2MapToken
type OAuth2MapToken struct {
	AccessToken string          `json:"access_token"`
	DataJson    json.RawMessage `json:"datajson"`
	CreateDate  time.Time       `json:"create_date"`
	ExpireDate  time.Time       `json:"expire_date"`
	Oauth2Token *oauth2.Token   `json:"-"`
	ProfileJson json.RawMessage `json:"profile_json"`
}

//swagger:model AuthTypeList
type AuthTypeList struct {
	Count int `json:"count"`
	Data  map[string]struct {
		Internal struct {
			// example: true
			Enable bool `json:"enable"`
			// example: internal
			Name string `json:"name"`
			// example: 1
			Position int `json:"position"`
			// example: internal
			Type string `json:"type"`
		} `json:"internal"`
		Oauth2 []struct {
			// example: true
			Enable bool `json:"enable"`
			// example: Google OAuth
			Name string `json:"name"`
			// example: 2
			Position int `json:"position"`
			// example: https://lh3.googleusercontent.com/j2SG2D_31o6aFxudKxHoXpTbMZIfA8afeRbxAM-PH9jFJPbj20YN0RUOSiikGWInnMP68lSmWG7j8qM-FGq6Y3bdSBPIwyPe7wPF=w96-rj-sc0xf8f9fa
			ProviderImage string `json:"provider_image"`
			// example: Google
			ProviderName string `json:"provider_name"`
			// example: oauth
			Type string `json:"type"`
			// example: /api/v3/oauth2/redirect/google
			URL string `json:"url"`
			// example: false
			EnableAutoRedirect bool `json:"auto_redirect"`
		} `json:"oauth2"`
	} `json:"data"`
	Message string `json:"message"`
}
