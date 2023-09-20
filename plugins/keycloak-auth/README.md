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
1. Copy the example source code in `CustomGoPlugin.go` file in the current directory and place it in the `go/src/CustomGoPlugin.go` file.

2. Copy the `auth.env` file to the `tyk/middleware/` directory and set the environment variable values.

3. Execute the make command to compile the plugin:
```
make build
```

4. Setup your API

In this example we will be using `http://httpbin.org/` as the upstream target. It's essentially a simple HTTP mock request and response service.

a. Enable `detailed_tracing` in your API definition (not required)

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