package flow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func OauthFlow(issuer, clientID, flow string) (string, error) {
	var provider *OauthProvider
	var token string
	provider, err := initializeOauthProvider(issuer)
	if err != nil {
		return "", err
	}
	switch flow {
	case "device":
		token, err = GetDeviceFlowToken(provider, clientID)
	default:
		return "", fmt.Errorf("unsupported oauth flow: %s", flow)
	}
	if err != nil {
		return "", err
	} else {
		return token, nil
	}

}

func initializeOauthProvider(url string) (*OauthProvider, error) {
	var provider *OauthProvider
	res, err := http.Get(url + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, &provider); err != nil {
		return nil, err
	}
	return provider, nil
}

func GetDeviceFlowToken(provider *OauthProvider, clientID string) (string, error) {

	values := url.Values{}
	values.Add("client_id", clientID)
	values.Add("scope", "openid email")

	resp, err := http.PostForm(provider.DeviceAuthEndpoint, values)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %s ", resp.Status)
	}
	var dr DeviceResp
	if err := json.Unmarshal(b, &dr); err != nil {
		return "", fmt.Errorf("error while unmarshling token: %s", err)
	}
	uric := dr.VerificationURIComplete
	uri := dr.VerificationURI
	if uri == "" {
		uri = dr.VerificationURI
	}
	fmt.Printf("\nOpen link : %s in browser and enter verification code %s\n", uri, dr.UserCode)
	fmt.Printf("\nOr open link : %s directly in the browser\n", uric)

	fmt.Printf("\nCode will be valid for %d seconds\n", dr.ExpiresIn)

	for {
		values := url.Values{}
		values.Add("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		values.Add("client_id", clientID)
		values.Add("device_code", dr.DeviceCode)
		values.Add("scope", "openid email")

		resp, err := http.PostForm(provider.TokenEndpoint, values)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		otr := OIDCTokenResponse{}
		if err := json.Unmarshal(b, &otr); err != nil {
			return "", err
		}

		if otr.AccessToken != "" {
			fmt.Println("\nTokens received!")
			val, err := json.MarshalIndent(otr, "", " ")
			if err != nil {
				return "", err
			}
			return string(val), nil

		}
		switch otr.Error {
		case "authorization_pending":
			fmt.Printf("\n debug: authorization request is still pending as the you have not completed authentication. sleeping for interval: %d\n", dr.Interval)
			time.Sleep(time.Duration(dr.Interval) * time.Second)
		case "slow_down":
			time.Sleep(time.Duration(dr.Interval)*time.Second + 5*time.Second)
		case "access_denied":
			return "", fmt.Errorf("the authorization request was denied: %s", otr.Error)
		case "expired_token":
			return "", fmt.Errorf("device_code has expired as it is older than: %d", dr.ExpiresIn)
		default:
			return "", fmt.Errorf("unexpected error in the device flow: %s", otr.Error)
		}
	}
}
