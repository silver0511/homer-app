package auth

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/sipcapture/homer-app/config"
	"github.com/sipcapture/homer-app/model"
	"github.com/sipcapture/homer-app/utils/logger"
)

// jwt token claims which contains info regarding user
type JwtUserClaim struct {
	UserName        string `json:"username"`
	UserAdmin       bool   `json:"useradmin"`
	UserGroup       string `json:"usergroup"`
	ExternalAuth    bool   `json:"externalauth"`
	ExternalProfile string `json:"externaltype"`
	DisplayName     string `json:"displayname"`
	Avatar          string `json:"avatar"`
	jwt.StandardClaims
}

func Token(user model.TableUser) (string, error) {

	tNow := time.Now()
	tUTC := tNow

	newTUTC := tUTC.Add(time.Duration(config.Setting.AUTH_SETTINGS.AuthTokenExpire) * time.Minute)

	// Set custom claims
	claims := &JwtUserClaim{
		user.UserName,
		user.IsAdmin,
		user.UserGroup,
		user.ExternalAuth,
		user.ExternalProfile,
		user.FirstName + " " + user.LastName,
		user.Avatar,
		jwt.StandardClaims{
			ExpiresAt: newTUTC.Unix(),
		},
	}

	logger.Debug("Current time : ", tNow)
	logger.Debug("Local time : ", tUTC)
	logger.Debug("Expire Local time : ", newTUTC)

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(config.Setting.AUTH_SETTINGS.JwtSecret))
	if err != nil {
		return "", err
	}

	return t, nil
}
