package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	uuid "github.com/satori/go.uuid"
	"github.com/sipcapture/homer-app/config"
	"github.com/sipcapture/homer-app/utils/heputils"
	"github.com/sipcapture/homer-app/utils/httpauth"
	"github.com/sipcapture/homer-app/utils/logger"

	"github.com/sipcapture/homer-app/auth"
	"github.com/sipcapture/homer-app/model"
	"github.com/sipcapture/homer-app/utils/ldap"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	ServiceConfig
	LdapClient *ldap.LDAPClient
	HttpAuth   *httpauth.Client
}

// this method gets all users from database
func (us *UserService) GetUser(UserName string, isAdmin bool) ([]*model.TableUser, int, error) {

	var user []*model.TableUser
	var sqlWhere = make(map[string]interface{})

	if !isAdmin {
		sqlWhere = map[string]interface{}{"username": UserName}
	}

	if err := us.Session.Debug().Table("users").Where(sqlWhere).Find(&user).Error; err != nil {
		return user, 0, err
	}

	return user, len(user), nil
}

// this method gets all users from database
func (us *UserService) GetUserByUUID(GUID, UserName string) ([]*model.TableUser, int, error) {

	var user []*model.TableUser
	var sqlWhere = make(map[string]interface{})

	sqlWhere = map[string]interface{}{"username": UserName, "guid": GUID}

	if err := us.Session.Debug().Table("users").Where(sqlWhere).Find(&user).Error; err != nil {
		return user, 0, err
	}

	return user, len(user), nil
}

/* get all */
func (us *UserService) GetGroups() (string, error) {

	count := len(config.Setting.MAIN_SETTINGS.UserGroups)
	reply := gabs.New()
	reply.Set(count, "count")
	reply.Set(config.Setting.MAIN_SETTINGS.UserGroups, "data")
	return reply.String(), nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (us *UserService) CreateNewUser(user *model.TableUser) error {

	user.CreatedAt = time.Now()

	if user.Password == "" {
		return errors.New("empty password")
	}

	if !heputils.ElementRealExists(config.Setting.MAIN_SETTINGS.UserGroups, user.UserGroup) {
		logger.Error("create user with group that doesn't exist: ", user.UserGroup)
		return fmt.Errorf("the user group '%s' doesn't exist", user.UserGroup)
	}

	// lets generate hash from password
	password := []byte(user.Password)

	// Hashing the password with the default cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	user.Hash = string(hashedPassword)

	err = us.Session.Debug().Table("users").Create(&user).Error
	if err != nil {
		return err
	}
	return nil
}

// this method update user info in the database
// it doesn't check internally whether all the validation are applied or not
func (us *UserService) UpdateUser(user *model.TableUser, UserName string, isAdmin bool) error {

	// get new instance of user data source
	user.CreatedAt = time.Now()
	oldRecord := model.TableUser{}

	var sqlWhere = make(map[string]interface{})

	if !isAdmin {
		sqlWhere = map[string]interface{}{"guid": user.GUID, "username": UserName}
	} else {
		sqlWhere = map[string]interface{}{"guid": user.GUID}
	}

	if us.Session.Where(sqlWhere).Find(&oldRecord).RecordNotFound() {
		return fmt.Errorf("the user with id '%s' was not found", user.GUID)
	}

	if !heputils.ElementRealExists(config.Setting.MAIN_SETTINGS.UserGroups, user.UserGroup) {
		logger.Error("create user with group that doesn't exist: ", user.UserGroup)
		return fmt.Errorf("the user group '%s' doesn't exist", user.UserGroup)
	}

	if user.Password != "" {
		password := []byte(user.Password)
		// Hashing the password with the default cost of 10
		hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.Hash = string(hashedPassword)
	} else {
		user.Hash = oldRecord.Hash
	}
	if !isAdmin {
		err := us.Session.Debug().Table("users").Model(&model.TableUser{}).Where(sqlWhere).Update(model.TableUser{Email: user.Email, FirstName: user.FirstName,
			LastName: user.LastName, Department: user.Department, Hash: user.Hash, CreatedAt: user.CreatedAt}).Error
		if err != nil {
			return err
		}
	} else {
		err := us.Session.Debug().Table("users").Model(&model.TableUser{}).Where(sqlWhere).Update(model.TableUser{UserName: user.UserName,
			PartId: user.PartId, Email: user.Email, FirstName: user.FirstName, LastName: user.LastName, Department: user.Department, UserGroup: user.UserGroup,
			Hash: user.Hash, CreatedAt: user.CreatedAt}).Error
		if err != nil {
			return err
		}
	}

	return nil

}

// this method deletes user in the database
// it doesn't check internally whether all the validation are applied or not
func (us *UserService) DeleteUser(user *model.TableUser) error {

	// get new instance of user data source
	newUser := model.TableUser{}

	if us.Session.Where("guid =?", user.GUID).Find(&newUser).RecordNotFound() {
		return fmt.Errorf("the user with id '%s' was not found", user.GUID)
	}
	err := us.Session.Debug().Where("guid =?", user.GUID).Delete(&model.TableUser{}).Error
	if err != nil {
		return err
	}
	return nil
}

// this method is used to login the user
// it doesn't check internally whether all the validation are applied or not
func (us *UserService) LoginUser(username, password string) (string, model.TableUser, error) {
	userData := model.TableUser{}

	switch {
	case us.LdapClient != nil:

		ok, isAdmin, user, err := us.LdapClient.Authenticate(username, password)
		if err != nil {
			/* second try after reconnect */
			ok, isAdmin, user, err = us.LdapClient.Authenticate(username, password)
			if err != nil {
				errorString := fmt.Sprintf("Error authenticating user %s: %+v", username, err)
				return "", userData, errors.New(errorString)
			}
		}

		if !ok {
			return "", userData, errors.New("authenticating failed for user")
		}

		userData.UserName = username
		userData.Id = int(hashString(username))
		hash := md5.Sum([]byte(username))
		userData.GUID = hex.EncodeToString(hash[:])
		userData.Password = password
		userData.FirstName = username
		userData.IsAdmin = isAdmin
		userData.ExternalAuth = true
		if val, ok := user["dn"]; ok {
			userData.UserGroup = val
		}

		logger.Debug("LDAP Auth: user: ", userData.UserName, " userID: ", userData.Id, " UserGUID: ", userData.GUID)

		userid := username

		// Microsoft AD implementations require DN for 1.2.840.113556.1.4.1941 recursive group query
		if us.LdapClient.UseDNForGroupSearch {
			userid = user["dn"]
		}

		groups, err := us.LdapClient.GetGroupsOfUser(userid)
		//fmt.Println("LDAP returned groups: ", groups)

		if err != nil {
			logger.Error("Couldn't get any group for user ", username, ": ", err)
			if !us.LdapClient.UserMode && !us.LdapClient.AdminMode {
				return "", userData, errors.New("couldn't fetch any LDAP group and membership is required for login")
			}
		} else {
			logger.Debug("Found groups for user ", username, ": ", groups)
			// ElementExists returns true if the given slice is empty, so we explicitly check that here
			// to prevent users with no groups from becoming admins
			if len(groups) > 0 && heputils.ElementExists(groups, us.LdapClient.AdminGroup) {
				logger.Debug("User ", username, " is a member of the admin group ", us.LdapClient.AdminGroup)
				userData.IsAdmin = true
			} else if len(groups) > 0 && heputils.ElementExists(groups, us.LdapClient.UserGroup) {
				logger.Debug("User ", username, " is a member of the user group ", us.LdapClient.UserGroup)
				userData.IsAdmin = false
			} else {
				if !userData.IsAdmin && us.LdapClient.UserMode {
					logger.Debug("User ", username, " didn't match any group but still logged in as USER because UserMode is set to true.")
					userData.UserGroup = "user"
				}
				if userData.IsAdmin {
					logger.Debug("User ", username, " didn't match any group but still logged in as ADMIN because AdminMode is set to true.")
					userData.UserGroup = "admin"
				}
				if !userData.IsAdmin && !us.LdapClient.UserMode {
					return "", userData, errors.New("failed group match. Group membership is required for login because AdminMode and UserMode are false")
				}
			}
		}
	case us.HttpAuth != nil:
		response, err := us.HttpAuth.Authenticate(username, password)
		if err != nil {
			return "", userData, errors.New("password is not correct")
		}
		if !response.Auth {
			return "", userData, errors.New("password is not correct")
		}
		userData = response.Data
		userData.IsAdmin = false
		userData.ExternalAuth = false
		if userData.UserGroup != "" && strings.Contains(strings.ToLower(userData.UserGroup), "admin") {
			userData.IsAdmin = true
		}
	default:
		if err := us.Session.Debug().Table("users").Where("username =?", username).Find(&userData).Error; err != nil {
			return "", userData, errors.New("user is not found")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(userData.Hash), []byte(password)); err != nil {
			return "", userData, errors.New("password is not correct")
		}

		/* check admin or not */
		userData.IsAdmin = false
		userData.ExternalAuth = false
		if userData.UserGroup != "" && strings.Contains(strings.ToLower(userData.UserGroup), "admin") {
			userData.IsAdmin = true
		}
	}

	if config.Setting.MAIN_SETTINGS.EnableGravatar && userData.Email != "" {
		hash := md5.Sum([]byte(userData.Email))
		userData.Avatar = fmt.Sprintf(config.Setting.MAIN_SETTINGS.GravatarUrl, hex.EncodeToString(hash[:]))
	}

	token, err := auth.Token(userData)
	return token, userData, err
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// this method gets all users from database
func (us *UserService) GetAuthTypeList() ([]byte, error) {

	var userGlobalSettings = model.TableGlobalSettings{}

	replyFinal := gabs.New()

	replyInternal := gabs.New()
	replyInternal.Set("Internal", "name")
	replyInternal.Set("internal", "type")
	replyInternal.Set(1, "position")

	if config.Setting.MAIN_SETTINGS.DefaultAuth == "internal" {
		replyInternal.Set(true, "enable")
	} else {
		replyInternal.Set(false, "enable")
	}

	replyLdap := gabs.New()
	replyLdap.Set("LDAP", "name")
	replyLdap.Set("ldap", "type")
	replyLdap.Set(2, "position")

	if config.Setting.MAIN_SETTINGS.DefaultAuth == "ldap" {
		replyLdap.Set(true, "enable")
	} else {
		replyLdap.Set(false, "enable")
	}

	if config.Setting.OAUTH2_SETTINGS.Enable {
		replyOauth := gabs.New()
		replyOauth.Set(config.Setting.OAUTH2_SETTINGS.ProjectID, "name")
		replyOauth.Set(config.Setting.OAUTH2_SETTINGS.ServiceProviderName, "provider_name")
		replyOauth.Set(config.Setting.OAUTH2_SETTINGS.UrlToServiceRedirect+"/"+config.Setting.OAUTH2_SETTINGS.ServiceProviderName, "url")
		replyOauth.Set(config.Setting.OAUTH2_SETTINGS.ServiceProviderImage, "provider_image")
		replyOauth.Set(config.Setting.OAUTH2_SETTINGS.EnableAutoRedirect, "auto_redirect")
		replyOauth.Set("oauth2", "type")
		replyOauth.Set(3, "position")
		replyOauth.Set(true, "enable")
		replyFinal.ArrayAppend(replyOauth.Data(), "oauth2")
	}

	replyFinal.Set(replyInternal.Data(), "internal")
	replyFinal.Set(replyLdap.Data(), "ldap")

	userGlobalSettings = model.TableGlobalSettings{
		Id:         1,
		GUID:       uuid.NewV4().String(),
		PartId:     10,
		Category:   "system",
		Param:      "authtypes",
		Data:       json.RawMessage(replyFinal.String()),
		CreateDate: time.Now(),
	}

	oj := model.SuccessfulResponse{}
	oj.Data = userGlobalSettings.Data
	oj.Count = 1
	oj.Message = "all good"

	return json.Marshal(oj)
}

// this method is used to login the user
// it doesn't check internally whether all the validation are applied or not
func (us *UserService) LoginUserUsingOauthToken(oAuth2Object model.OAuth2MapToken) (string, model.TableUser, error) {

	userJsonData, _ := gabs.ParseJSON(oAuth2Object.ProfileJson)

	userData := model.TableUser{}

	logger.Debug("LoginUserUsingOauthToken json profile: ", string(oAuth2Object.ProfileJson))

	if userJsonData.Exists("email") {
		userData.Email = userJsonData.S("email").Data().(string)
		userData.UserName = userData.Email
		userData.Id = int(hashString(userData.UserName))
	}

	if userJsonData.Exists("family_name") {
		userData.LastName = userJsonData.S("family_name").Data().(string)
	}

	if userJsonData.Exists("given_name") {
		userData.FirstName = userJsonData.S("given_name").Data().(string)
	}

	if userJsonData.Exists("picture") {
		userData.Avatar = userJsonData.S("picture").Data().(string)
	}

	if config.Setting.OAUTH2_SETTINGS.EnableGravatar && userData.Email != "" {
		hash := md5.Sum([]byte(userData.Email))
		userData.Avatar = fmt.Sprintf(config.Setting.OAUTH2_SETTINGS.GravatarUrl, hex.EncodeToString(hash[:]))
	}

	if userJsonData.Exists("id") {

		s := (userJsonData.S("id").Data().(string))
		i, err := strconv.Atoi(s)
		if err == nil {
			userData.Id = i
		} else {
			logger.Error("bad ID size: ", s, i)

		}
	}

	hash := md5.Sum([]byte(userData.UserName))
	userData.GUID = hex.EncodeToString(hash[:])
	userData.ExternalAuth = true
	userData.UserGroup = "user"
	userData.IsAdmin = false

	token, err := auth.Token(userData)
	return token, userData, err
}

// this method gets all users from database
/*func (us *UserService) GetUserFromToken(userTokenProfile *auth.JwtUserClaim) (model.TableUser, error) {

	userProfile := model.TableUser{}

	userProfile.Id = 0
	userProfile.UserName = userTokenProfile.UserName
	userProfile.UserGroup = userTokenProfile.UserGroup
	userProfile.Email = userTokenProfile.ExternalProfile
	userProfile.LastName = userTokenProfile.DisplayName
	userProfile.Avatar = userTokenProfile.Avatar
	userProfile.IsAdmin = userTokenProfile.UserAdmin
	userProfile.ExternalAuth = userTokenProfile.ExternalAuth
	userProfile.ExternalProfile = userTokenProfile.ExternalProfile

	return userProfile, nil
}
*/

/* get all */
func (us *UserService) GetUserProfileFromToken(userTokenProfile *auth.JwtUserClaim) (string, error) {

	userProfile := model.UserProfile{}
	userProfile.UserName = userTokenProfile.UserName
	userProfile.UserGroup = userTokenProfile.UserGroup
	userProfile.ExternalAuth = userTokenProfile.ExternalAuth
	userProfile.DisplayName = userTokenProfile.DisplayName
	userProfile.Avatar = userTokenProfile.Avatar
	userProfile.UserAdmin = userTokenProfile.UserAdmin

	userProfile.ExternalProfile = userTokenProfile.ExternalProfile

	if !userTokenProfile.ExternalAuth {
		user, count, err := us.GetUser(userTokenProfile.UserName, false)
		if err == nil && count > 0 {
			userProfile.GUID = user[0].GUID
		}
	}

	data, _ := json.Marshal(userProfile)
	rows, _ := gabs.ParseJSON(data)
	count, _ := rows.ArrayCount()

	reply := gabs.New()
	reply.Set(count, "count")
	reply.Set(rows.Data(), "data")
	return reply.String(), nil
}
