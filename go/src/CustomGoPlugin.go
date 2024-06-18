package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/TykTechnologies/opentelemetry/trace"
	"github.com/TykTechnologies/tyk/apidef/oas"
	"github.com/TykTechnologies/tyk/ctx"
	"github.com/TykTechnologies/tyk/log"
	"github.com/TykTechnologies/tyk/user"
	"github.com/golang-jwt/jwt/v4"
)

var logger = log.Get()

// AddFooBarHeader adds custom "Foo: Bar" header to the request
func AddFooBarHeader(rw http.ResponseWriter, r *http.Request) {
	// We create a new span using the context from the incoming request.
	_, newSpan := trace.NewSpanFromContext(r.Context(), "", "GoPlugin_first-span")

	// Ensure that the span is properly ended when the function completes.
	defer newSpan.End()

	// Set a new name for the span.
	newSpan.SetName("AddFooBarHeader Function")

	// Set the status of the span.
	newSpan.SetStatus(trace.SPAN_STATUS_OK, "")

	r.Header.Add("X-SimpleHeader-Inject", "foo")
}

// AuthCheck Custom Auth, applies a rate limit of
// 2 per 10 given a token of "abc"
func AuthCheck(rw http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	_, newSpan := trace.NewSpanFromContext(r.Context(), "", "GoPlugin_custom-auth")
	defer newSpan.End()

	if token != "d3fd1a57-94ce-4a36-9dfe-679a8f493b49" && token != "3be61aa4-2490-4637-93b9-105001aa88a5" {
		newSpan.SetAttributes(trace.NewAttribute("auth", "failed"))
		newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "")

		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	newSpan.SetAttributes(trace.NewAttribute("auth", "success"))
	newSpan.SetStatus(trace.SPAN_STATUS_OK, "")

	session := &user.SessionState{
		Alias: token,
		Rate:  2,
		Per:   10,
		MetaData: map[string]interface{}{
			token: token,
		},
		KeyID: token,
	}

	ctx.SetSession(r, session, true)
}

// InjectMetadata Injects meta data from a token where the metadata key is "foo"
func InjectMetadata(rw http.ResponseWriter, r *http.Request) {
	session := ctx.GetSession(r)
	if session != nil {
		// Access session fields such as MetaData
		metaData := session.MetaData
		foo, ok := metaData["foo"].(string) // Type assert foo to string
		if !ok {
			// Handle the case where foo is not a string or foo does not exist
			logger.Error("Error: 'foo' is not a string or not found in metaData")
			return // or continue, depending on your error handling strategy
		}
		// Process metaData as needed
		r.Header.Add("X-Metadata-Inject", foo)
	}
}

// InjectConfigData Injects config data, both from an env variable and hard-coded
func InjectConfigData(rw http.ResponseWriter, r *http.Request) {
	oasDef := ctx.GetOASDefinition(r)

	// Extract the middleware section safely
	xTyk, ok := oasDef.Extensions["x-tyk-api-gateway"].(*oas.XTykAPIGateway)
	if !ok {
		logger.Println("Middleware extension is missing or invalid.")
		return
	}

	configKey := xTyk.Middleware.Global.PluginConfig.Data.Value["env-config-example"].(string)
	r.Header.Add("X-ConfigData-Config", configKey)
}

// MakeOutboundCall Injects config data, both from an env variable and hard-coded
func MakeOutboundCall(rw http.ResponseWriter, r *http.Request) {
	// Define the URL
	url := "https://httpbin.org/get"

	// Create a GET request
	response, err := http.Get(url)
	if err != nil {
		logger.Info("Error making GET request: %s\n", err)
		return
	}
	defer response.Body.Close()

	// Read the response body
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Info("Error reading response: %s\n", err)
		return
	}

	// Print the response body
	logger.Info(string(responseData))
}

func main() {}

// This will be run during Gateway startup
func init() {
	logger.Info("--- Go custom plugin init success! ---- ")
}

// IPRateLimit
func IPRateLimit(rw http.ResponseWriter, r *http.Request) {
	// Get the client IP address
	clientIP := getIP(r.RemoteAddr)

	logger.Info("Request came in from " + clientIP)

	// Create a Redis key for the IP address
	session := &user.SessionState{
		Alias: clientIP,
		Rate:  2,
		Per:   10,
		MetaData: map[string]interface{}{
			"token": clientIP,
		},
		KeyID: clientIP,
	}

	ctx.SetSession(r, session, true)
}

func getIP(remoteAddr string) string {
	if ip := strings.Split(remoteAddr, ":"); len(ip) > 0 {
		return ip[0]
	}
	return ""
}

// JWTValidate Validate a JWT signed using PS256/382/512
func JWTValidate(rw http.ResponseWriter, r *http.Request) {
	logger.Info("Request came in - custom auth")

	tokenString := r.Header.Get("Authorization")

	oasDef := ctx.GetOASDefinition(r)

	// Extract the middleware section safely
	xTyk, ok := oasDef.Extensions["x-tyk-api-gateway"].(*oas.XTykAPIGateway)
	if !ok {
		logger.Println("Middleware extension is missing or invalid.")
		return
	}

	publicKeyPEM := xTyk.Middleware.Global.PluginConfig.Data.Value["env-config-example"].(string)

	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
	if err != nil {
		logger.Info("Error parsing public key: %v", err)
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSAPSS); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		logger.Info("Error parsing token: %v", err)
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !token.Valid {
		logger.Info("Invalid token")
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	logger.Info("Token is valid")

	// Create a Redis key for the IP address
	session := &user.SessionState{
		Alias: token.Raw,
		Rate:  0,
		Per:   0,
		MetaData: map[string]interface{}{
			"token": token.Raw,
		},
		KeyID: token.Raw,
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.Info("Error casting claims")
	}

	// Example: Get "username" from the claims
	sub := claims["sub"].(string)
	logger.Info("sub: %s\n", sub)

	r.Header.Add("X-Injected-Sub", sub)

	ctx.SetSession(r, session, true)
}
