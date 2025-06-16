# CLI to execute Device Authorization Grant Flow

Modification of forked project to save a access token in config and reuse it if valid. If expired, use stored refresh token to get a new access token.

GitHub OAuth Identity Provider with Device Flow. Using Client ID : device-test-client - we can run the CLI as follows:

```
$ go run cmd/oauth/main.go -c device-test-client
```

The sample response will be like following:

```
Open link : https://github.com/login/device in browser and enter verification code APFN-HGMB

Code will be valid for 28800 seconds

Tokens received!
Received response: {
 "access_token": "xxxxx",
 "token_type": "bearer",
 "refresh_token": "xxxxx",
 "expires_in": 28800,
 "error": ""
}
```

Alternatively, if there is already a valid token, it will simply say:

```
Using cached token!
Received response: {
 "access_token": "xxxxx",
 "token_type": "bearer",
 "refresh_token": "xxxxx",
 "expires_at": "2025-06-16T17:34:17.543125+02:00"
}
```

Finally, if the token has already expired, it will attempt to get a new one via the refresh token:

```
Token expired, attempting to refresh...

Token refreshed successfully!
Received response: {
 "access_token": "xxxxx",
 "token_type": "bearer",
 "refresh_token": "xxxxx",
 "expires_at": "2025-06-16T17:42:02.077096+02:00"
}
```

Refer linked [blog](https://medium.com/@rishabhsvats/developing-golang-cli-to-test-device-authorization-grant-with-keycloak-6e0e6e6dfe82) for more details about Device authorization grant and implementation of this project.
