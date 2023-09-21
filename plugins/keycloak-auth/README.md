# Keycloak Custom Auth Example

> [!WARNING]
> Please keep in mind that this Custom Go Plugin is not production ready and should only be used to get started.

Current functionality of this Keycloak Custom Auth plugin:
1. Calls external IdPs to act on them
2. Able to create new access tokens using username and password from IdP
3. Stores in Redis the tokens with the refresh tokens
4. If the access token is not valid after introspection it searches for its refresh token in Redis, it asks for a new one to the IdP, and gives the new one to the user in a header
5. If there are multiple new request while the token is expired (R1 to RN):
    1. R1 access with expired `access_token`
    2. Plugin blocks Key replacing refresh token with `"old"` - `access_token:old` for example - `1111:old`
    3. R1 plugin asks for new `access_token`
        1. New Key `access_token:refresh_token - 2222:yyyyy`
        2. Plugin adds a new key called `access_token_exp:new_access_token - 1111-exp:2222`
            - This way RN can check for the new token having the expired one.
    7. RN checks if the key is blocked `"old"` if yes, it will loop looking for `access_token-exp`
    8. RN will get a response and the new token in a response header.

## How to test the Custom Plugin
1. Copy the example source code files `CustomGoPlugin.go` and `model.go` and place it in the `go/src/` directory. If you plan on using the Otel version, copy only the `CustomGoPluginOtel.go` file.

2. Copy the `auth.env` file to the `tyk/middleware/` directory and set the environment variable values.
```
OAUTH2_KEYCLOAK_INTROSPECT_ENDPOINT="http://host.docker.internal:8180/realms/tyk/protocol/openid-connect/token/introspect"
OAUTH2_KEYCLOAK_INTROSPECT_AUTHORIZATION="Basic base64('CLIENT_ID:CLIENT_SECRET')"
OAUTH2_KEYCLOAK_CLIENT_ID="CLIENT_ID"
OAUTH2_KEYCLOAK_CLIENT_SECRET="CLIENT_SECRET"
OAUTH2_KEYCLOAK_TOKEN_ENDPOINT="http://host.docker.internal:8180/realms/tyk/protocol/openid-connect/token"
```

3. Execute the make command to compile the plugin:
```
make build
```

4. Setup your API

In this example we will be using `http://httpbin.org/` as the upstream target. It's essentially a simple HTTP mock request and response service.

a. Follow [JWT and Keycloak with Tyk](https://tyk.io/docs/basic-config-and-security/security/authentication-authorization/json-web-tokens/jwt-keycloak/) documentation to set up your API with JWT/Keycloak authentication.

b. Enable `detailed_tracing` in your API definition if you plan on using Open Telemetry to debug/troubleshoot. 

> [!NOTE]
> If you have deployed Tyk version `v5.2.0+` and would like to use Open Telemetry to debug and troubleshoot.
```
{
    ...
    "detailed_tracing": true
    ...
}
```

b. Configure your `custom_middleware` in your API definition
```
"custom_middleware": {
    ...
    "pre": [{
        "disabled": false,
        "name": "OAuth2Introspect",
        "path": "/opt/tyk-gateway/middleware/CustomGoPlugin.so",
        "require_session": false,
        "raw_body_only": false
    }],
    "driver": "goplugin",
    ...
}
```

5. Send an API request

a. Send an API request using your Keycloak user credentials to retreive a `${BEARER_TOKEN}`. You will notice that the response containers a header `New_access_token`.

```
❯ curl --location 'http://localhost:8080/httpbin/json' -v \
--header 'user: ${USER}' \
--header 'password: ${PASSWORD}'

*   Trying 127.0.0.1:8080...
* Connected to localhost (127.0.0.1) port 8080 (#0)
> GET /httpbin/json HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/8.1.2
> Accept: */*
> user: ${USER}
> password: ${PASSWORD}
>
< HTTP/1.1 200 OK
< Access-Control-Allow-Credentials: true
< Access-Control-Allow-Origin: *
< Content-Length: 429
< Content-Type: application/json
< Date: Wed, 20 Sep 2023 19:52:45 GMT
< New_access_token: ${BEARER_TOKEN}
< Server: gunicorn/19.9.0
< X-Ratelimit-Limit: 0
< X-Ratelimit-Remaining: 0
< X-Ratelimit-Reset: 0
< X-Tyk-Trace-Id: 97430186c977570d4b891e2026ee84d6
<
{
  "slideshow": {
    "author": "Yours Truly",
    "date": "date of publication",
    "slides": [
      {
        "title": "Wake up to WonderWidgets!",
        "type": "all"
      },
      {
        "items": [
          "Why <em>WonderWidgets</em> are great",
          "Who <em>buys</em> WonderWidgets"
        ],
        "title": "Overview",
        "type": "all"
      }
    ],
    "title": "Sample Slide Show"
  }
}
```

b. Send an API request using the `${BEARER_TOKEN}` retrieved from the previous call.

```
❯ curl --location 'http://localhost:8080/httpbin/json' \
--header 'Authorization: Bearer ${BEARER_TOKEN}'
{
  "slideshow": {
    "author": "Yours Truly",
    "date": "date of publication",
    "slides": [
      {
        "title": "Wake up to WonderWidgets!",
        "type": "all"
      },
      {
        "items": [
          "Why <em>WonderWidgets</em> are great",
          "Who <em>buys</em> WonderWidgets"
        ],
        "title": "Overview",
        "type": "all"
      }
    ],
    "title": "Sample Slide Show"
  }
}
```

c. Using the `${BEARER_TOKEN}` you can send an API request to your instrospection endpoint to determine if the token is still active or not.

```
> curl --location 'http://localhost:8180/realms/tyk/protocol/openid-connect/token/introspect' \
--header 'Accept: application/json' \
--header 'Authorization: Basic base64(CLIENT_ID:CLIENT_SECRET)' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'token=${BEARER_TOKEN}'
{
    "exp": 1695328468,
    "iat": 1695328168,
    "jti": "000cdca9-a124-4ce3-9284-150e0978f208",
    "iss": "http://localhost:8180/realms/tyk",
    "aud": "account",
    "sub": "1d2d88f8-8b68-4b6b-883f-1af68b598612",
    "typ": "Bearer",
    "azp": "tykjwt",
    "preferred_username": "service-account-tykjwt",
    "email_verified": false,
    "acr": "1",
    "allowed-origins": [
        "/*"
    ],
    "realm_access": {
        "roles": [
            "default-roles-tyk",
            "offline_access",
            "uma_authorization"
        ]
    },
    "resource_access": {
        "account": {
            "roles": [
                "manage-account",
                "manage-account-links",
                "view-profile"
            ]
        }
    },
    "scope": "openid email profile",
    "clientHost": "172.17.0.1",
    "clientAddress": "172.17.0.1",
    "client_id": "tykjwt",
    "username": "service-account-tykjwt",
    "active": true
}
```