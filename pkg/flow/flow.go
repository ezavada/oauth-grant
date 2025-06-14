package flow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubDeviceAuthEndpoint = "https://github.com/login/device/code"
	githubTokenEndpoint      = "https://github.com/login/oauth/access_token"
)

func OauthFlow(clientID string) (string, error) {
	return GetGitHubDeviceFlowToken(clientID)
}

func GetGitHubDeviceFlowToken(clientID string) (string, error) {
	values := url.Values{}
	values.Add("client_id", clientID)
	values.Add("scope", "read:user user:email") // GitHub specific scopes

	req, err := http.NewRequest("POST", githubDeviceAuthEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
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
		return "", fmt.Errorf("error while unmarshaling device response: %s", err)
	}

	uri := dr.VerificationURI
	uric := dr.VerificationURIComplete
	if uri == "" {
		uri = dr.VerificationURI
	}

	fmt.Printf("\nOpen link : %s in browser and enter verification code %s\n", uri, dr.UserCode)
	fmt.Printf("\nOr open link : %s directly in the browser\n", uric)
	fmt.Printf("\nCode will be valid for %d seconds\n", dr.ExpiresIn)

	for {
		values := url.Values{}
		values.Add("client_id", clientID)
		values.Add("device_code", dr.DeviceCode)
		values.Add("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		req, err := http.NewRequest("POST", githubTokenEndpoint, strings.NewReader(values.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		var tokenResp struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Error       string `json:"error"`
		}

		if err := json.Unmarshal(b, &tokenResp); err != nil {
			return "", err
		}

		if tokenResp.AccessToken != "" {
			fmt.Println("\nToken received!")
			val, err := json.MarshalIndent(tokenResp, "", " ")
			if err != nil {
				return "", err
			}
			return string(val), nil
		}

		switch tokenResp.Error {
		case "authorization_pending":
			fmt.Printf("\nAuthorization request is still pending. Waiting for %d seconds...\n", dr.Interval)
			time.Sleep(time.Duration(dr.Interval) * time.Second)
		case "slow_down":
			time.Sleep(time.Duration(dr.Interval)*time.Second + 5*time.Second)
		case "access_denied":
			return "", fmt.Errorf("the authorization request was denied")
		case "expired_token":
			return "", fmt.Errorf("device_code has expired")
		default:
			return "", fmt.Errorf("unexpected error in the device flow: %s", tokenResp.Error)
		}
	}
}
