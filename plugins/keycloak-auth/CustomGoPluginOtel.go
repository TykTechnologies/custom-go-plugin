package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TykTechnologies/opentelemetry/trace"
	"github.com/TykTechnologies/tyk/config"
	"github.com/TykTechnologies/tyk/log"
	"github.com/TykTechnologies/tyk/storage"
	"github.com/joho/godotenv"
)

const pluginDefaultKeyPrefix = "Plugin-data:"

var (
	introspectionClient *http.Client
	introspectLogger    = log.Get()

	introspectionEndpoint    = ""
	authorizationHeaderName  = "Authorization"
	authorizationHeaderValue = ""
	clientID                 = ""
	clientSecret             = ""
	tokenEndpoint            = ""

	// Global redis variables
	conf  config.Config
	rc    *storage.RedisController
	store = storage.RedisCluster{KeyPrefix: pluginDefaultKeyPrefix}
)

type StringOrArrString []string

// https://tools.ietf.org/html/rfc7662
type IntrospectResponse struct {
	// active REQUIRED.  Boolean indicator of whether or not the presented token
	//      is currently active.  The specifics of a token's "active" state
	//      will vary depending on the implementation of the authorization
	//      server and the information it keeps about its tokens, but a "true"
	//      value return for the "active" property will generally indicate
	//      that a given token has been issued by this authorization server,
	//      has not been revoked by the resource owner, and is within its
	//      given time window of validity (e.g., after its issuance time and
	//      before its expiration time).  See Section 4 for information on
	//      implementation of such checks.
	Active bool `json:"active"`
	// scope OPTIONAL.  A JSON string containing a space-separated list of
	//      scopes associated with this token, in the format described in
	//      Section 3.3 of OAuth 2.0 [RFC6749].
	Scope *string `json:"scope,omitempty"`
	// client_id OPTIONAL.  Client identifier for the OAuth 2.0 client that
	//      requested this token.
	ClientID *string `json:"client_id"`
	// username OPTIONAL.  Human-readable identifier for the resource owner who
	//      authorized this token.
	Username *string `json:"username"`
	// token_type OPTIONAL.  Type of the token as defined in Section 5.1 of OAuth
	//      2.0 [RFC6749].
	TokenType *string `json:"token_type"`
	// exp OPTIONAL.  Integer timestamp, measured in the number of seconds
	//      since January 1 1970 UTC, indicating when this token will expire,
	//      as defined in JWT [RFC7519].
	Exp *int64 `json:"exp"`
	// iat OPTIONAL.  Integer timestamp, measured in the number of seconds
	//      since January 1 1970 UTC, indicating when this token was
	//      originally issued, as defined in JWT [RFC7519].
	Iat *int64 `json:"iat"`
	// nbf OPTIONAL.  Integer timestamp, measured in the number of seconds
	//      since January 1 1970 UTC, indicating when this token is not to be
	//      used before, as defined in JWT [RFC7519].
	Nbf *int64 `json:"nbf"`
	// sub OPTIONAL.  Subject of the token, as defined in JWT [RFC7519].
	//      Usually a machine-readable identifier of the resource owner who
	//      authorized this token.
	Sub *string `json:"sub"`
	// aud OPTIONAL.  Service-specific string identifier or list of string
	//      identifiers representing the intended audience for this token, as
	//      defined in JWT [RFC7519].
	Aud *StringOrArrString `json:"aud"`
	// iss OPTIONAL.  String representing the issuer of this token, as
	//      defined in JWT [RFC7519].
	Iss *string `json:"iss"`
	// jti OPTIONAL.  String identifier for the token, as defined in JWT
	//      [RFC7519].
	Jti *string `json:"jti"`
}
type OIDCToken struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not_before_policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

func (s *StringOrArrString) UnmarshalJSON(data []byte) error {
	if len(data) > 1 && data[0] == '[' {
		var obj []string
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		*s = StringOrArrString(obj)
		return nil
	}

	var obj string
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*s = StringOrArrString([]string{obj})
	return nil
}

// Function for access token introspection - checking if the token is valid and doing the logic for the correct flow
func OAuth2Introspect(w http.ResponseWriter, r *http.Request) {
	introspectLogger.Info("Start custom plugin")
	ctx, newSpan := trace.NewSpanFromContext(r.Context(), "", "GoPlugin_OAuth2Introspect")

	// Ensure that the span is properly ended when the function completes.
	defer newSpan.End()

	bearerToken := accessTokenFromRequest(r)
	//Checking if the bearerToken is added or if there is a need to create a new access_token + refresh_token using user and password from Keycloak
	if bearerToken == "" {
		user := getUser(r)
		password := getPassword(r)
		if user != "" && password != "" {
			//Creating a brandnew token for the user with username and password from Keycloak
			brandNewAccessToken(ctx, user, password, w, r)

			// Set span status
			newSpan.SetStatus(trace.SPAN_STATUS_OK, "")

			return
		} else {
			// Set span status
			newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "")

			introspectLogger.Info("no bearer token found in request")
			writeUnauthorized(w)
			return
		}
	}

	data := url.Values{}
	data.Set("token", bearerToken)
	data.Set("token_type_hint", "access_token")
	introspectionReq, err := http.NewRequest(http.MethodPost, introspectionEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		introspectLogger.Error("unable to create new request %s", err.Error())
		writeInternalServerError(w)
		return
	}
	introspectionReq.Header.Add(authorizationHeaderName, authorizationHeaderValue)
	introspectionReq.Header.Add("content-length", strconv.Itoa(len(data.Encode())))
	introspectionReq.Header.Add("content-type", "application/x-www-form-urlencoded")
	introspectionRes, err := introspectionClient.Do(introspectionReq)
	if err != nil {
		introspectLogger.Error("tyk cannot connect to the authorization server %s\n", err.Error())
		writeInternalServerError(w)
		return
	}
	if introspectionRes.StatusCode == http.StatusUnauthorized {
		introspectLogger.Error("tyk is not authorized to perform introspection")
		writeInternalServerError(w)
		return
	}
	defer introspectionRes.Body.Close()

	body, err := ioutil.ReadAll(introspectionRes.Body)
	if err != nil {
		introspectLogger.Error("unable to read response body from authorization server %s", err.Error())
		writeInternalServerError(w)
		return
	}

	irObj := &IntrospectResponse{}
	err = json.Unmarshal(body, irObj)
	if err != nil {
		introspectLogger.Error("unable to read json response from authorization server %s", err.Error())
		writeInternalServerError(w)
		return
	}
	//Checks if the access_token is inactive
	if irObj.Active == false {
		introspectLogger.Info("access_token is inactive")
		refreshToken, err := tykGetData(bearerToken)
		if err != nil {
			fmt.Println("Error:", err)
		}
		newAccessToken := ""
		//checks if the old token was flagged, flags it if not and asks for a new token
		if refreshToken != "old" {
			newAccessTokenRequest(w, r)
			tykStoreData(bearerToken, "old")
			introspectLogger.Info("Added old into bearertoken")
		} else {
			fmt.Println("checking for the exp")
			bearerTokenExp, err := tykGetData(bearerToken + "-exp")
			fmt.Println(bearerTokenExp)
			for err != nil {
				bearerTokenExp, err = tykGetData(bearerToken + "-exp")
				fmt.Println("Error:", err)
			}
			fmt.Println("Loop exited.")
			newAccessToken, err = tykGetData(bearerToken + "-exp")
			for err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println(newAccessToken)
			r.Header.Set(authorizationHeaderName, newAccessToken)
			w.Header().Set("new_access_token", newAccessToken)
		}
	}

}

// Function for new access_token when there is an available refresh token
func newAccessTokenRequest(w http.ResponseWriter, r *http.Request) {
	bearerToken := accessTokenFromRequest(r)
	// Checks if there is a Refresh Token in redis matching the inactive access_token
	// If there is no Refresh Token then it uses the hardcoded one.
	refreshTokenRedis, err := tykGetData(bearerToken)
	introspectLogger.Info(refreshTokenRedis)
	if err != nil {
		return
	}
	response, err := http.PostForm(tokenEndpoint, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshTokenRedis},
		"client_id":     {clientID},
		"client_secret": {clientSecret}})

	if err != nil {
		introspectLogger.Error("unable to create new request %s", err.Error())
		writeInternalServerError(w)
		return
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		introspectLogger.Error("unable to read response body from authorization server %s", err.Error())
		writeInternalServerError(w)
		return
	}
	irObj := &OIDCToken{}
	err = json.Unmarshal(body, &irObj)
	if err != nil {
		introspectLogger.Error("unable to read json response from authorization server %s", err.Error())
		return
	}
	r.Header.Set(authorizationHeaderName, irObj.AccessToken)
	w.Header().Set("new_access_token", irObj.AccessToken)
	//It will save the new access_token and the refresh_token assigned to it in Redis
	introspectLogger.Error("new token")
	tykStoreData(irObj.AccessToken, irObj.RefreshToken)
	//It will flag the old access token as -exp
	tykStoreData(bearerToken+"-exp", irObj.AccessToken)
	TokenRead, err := tykGetData(bearerToken + "-exp")
	introspectLogger.Info(err)
	introspectLogger.Info(TokenRead)

}

// Function to get a new access token using a username and password from Keycloak
func brandNewAccessToken(ctx context.Context, user string, password string, w http.ResponseWriter, r *http.Request) {
	// Create a new span using the context passed from the previous function.
	ctx, newSpan := trace.NewSpanFromContext(ctx, "", "GoPlugin_brandNewAccessToken")
	defer newSpan.End()

	response, err := http.PostForm(tokenEndpoint, url.Values{
		"scope":         {"openid"},
		"grant_type":    {"password"},
		"username":      {user},
		"password":      {password},
		"client_id":     {clientID},
		"client_secret": {clientSecret}})

	if err != nil {
		// Set an attribute on the new span.
		newSpan.SetAttributes(trace.NewAttribute("token.endpoint.post.request", err))

		// Set span status
		newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "")
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		// Set an attribute on the new span.
		newSpan.SetAttributes(trace.NewAttribute("response.body.reader", err))

		// Set span status
		newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "")
	}

	irObj := &OIDCToken{}
	err = json.Unmarshal(body, &irObj)
	if err != nil {
		// Set an attribute on the new span.
		newSpan.SetAttributes(trace.NewAttribute("oidc.token.json.unmarshal", err))

		// Set span status
		newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "")

		introspectLogger.Error("unable to read json response from authorization server %s", err.Error())
		return
	}
	//It will save the new access_token and the refresh_token assigned to it in Redis
	r.Header.Set(authorizationHeaderName, irObj.AccessToken)
	w.Header().Set("new_access_token", irObj.AccessToken)
	tykStoreData(irObj.AccessToken, irObj.RefreshToken)
}

func accessTokenFromRequest(r *http.Request) string {

	auth := r.Header.Get(authorizationHeaderName)
	split := strings.SplitN(auth, " ", 2)
	if len(split) != 2 || !strings.EqualFold(split[0], "bearer") {
		if err := r.ParseMultipartForm(1 << 20); err != nil && err != http.ErrNotMultipart {
			return ""
		}
		return r.Form.Get("access_token")
	}

	return split[1]
}

func getUser(r *http.Request) string {
	user := r.Header.Get("user")
	if user == "" {
		return ""
	}
	return user
}
func getPassword(r *http.Request) string {
	password := r.Header.Get("password")
	if password == "" {
		return ""
	}
	return password
}

func writeUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
}

func writeInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
}

func tykStoreData(key, value string) {
	ttl := int64(1000)
	store.SetKey(key, value, ttl)
}

func tykGetData(key string) (string, error) {
	val, err := store.GetKey(key)
	return val, err
}

// Checks if the refresh token is no longer valid and it was set to "old"
func refreshTokenOld(w http.ResponseWriter, r *http.Request) bool {
	introspectLogger.Info("Inactive Token")
	bearerToken := accessTokenFromRequest(r)
	refreshTokenRedis, err := tykGetData(bearerToken)
	introspectLogger.Info(refreshTokenRedis)
	if refreshTokenRedis == "old" {
		introspectLogger.Info("Already changed for Old")
		return true
	} else {
		return false
	}
	if err != nil {
		//handle postform error
	}
	return false
}

func establishRedisConnection() {
	// Retrieve global configs
	conf = config.Global()

	// Create a Redis Controller, which will handle the Redis connection for the storage
	rc = storage.NewRedisController(context.Background())

	// Create a storage object, which will handle Redis operations using "apikey-" key prefix
	store = storage.RedisCluster{KeyPrefix: pluginDefaultKeyPrefix, HashKeys: conf.HashKeys, RedisController: rc}

	// Perform Redis connection
	go rc.ConnectToRedis(context.Background(), nil, &conf)
	for i := 0; i < 5; i++ { // max 5 attempts - should only take 2
		time.Sleep(30 * time.Millisecond)
		if rc.Connected() {
			introspectLogger.Info("Redis Controller connected")
			break
		}
		introspectLogger.Warn("Redis Controller not connected, will retry")
	}

	// Error handling Redis connection
	if !rc.Connected() {
		introspectLogger.Warn("Could not connect to storage")
		panic("Couldn't esetablished a connection to redis")
	}
}

func main() {}

func init() {
	introspectLogger.Info("--- Keycloak Auth ---- ")
	introspectLogger.Info("--- Go custom plugin init success! ---- ")

	introspectionClient = &http.Client{}
	establishRedisConnection()

	err := godotenv.Load("/opt/tyk-gateway/middleware/auth.env")
	if err != nil {
		introspectLogger.Info("Error loading .env file")
	}
	introspectionEndpoint = os.Getenv("OAUTH2_KEYCLOAK_INTROSPECT_ENDPOINT")
	authorizationHeaderValue = os.Getenv("OAUTH2_KEYCLOAK_INTROSPECT_AUTHORIZATION")
	clientID = os.Getenv("OAUTH2_KEYCLOAK_CLIENT_ID")
	clientSecret = os.Getenv("OAUTH2_KEYCLOAK_CLIENT_SECRET")
	tokenEndpoint = os.Getenv("OAUTH2_KEYCLOAK_TOKEN_ENDPOINT")
}
