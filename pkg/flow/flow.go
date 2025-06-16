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

func refreshToken(clientID string, refreshToken string) (*TokenConfig, error) {
	values := url.Values{}
	values.Add("client_id", clientID)
	values.Add("grant_type", "refresh_token")
	values.Add("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", githubTokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}

	if err := json.Unmarshal(b, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("error refreshing token: %s", tokenResp.Error)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token received in refresh response")
	}

	config := &TokenConfig{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	if err := SaveTokenConfig(*config); err != nil {
		fmt.Printf("Warning: Failed to save refreshed token configuration: %v\n", err)
	}

	return config, nil
}

func GetGitHubDeviceFlowToken(clientID string) (string, error) {
	// First check if we have a valid token in the config
	if config, err := LoadTokenConfig(); err == nil && config != nil {
		// If token is expired but we have a refresh token, try to refresh it
		if time.Now().After(config.ExpiresAt) && config.RefreshToken != "" {
			fmt.Println("\nToken expired, attempting to refresh...")
			newConfig, err := refreshToken(clientID, config.RefreshToken)
			if err == nil && newConfig != nil {
				fmt.Println("\nToken refreshed successfully!")
				val, err := json.MarshalIndent(newConfig, "", " ")
				if err != nil {
					return "", err
				}
				return string(val), nil
			}
			fmt.Printf("\nFailed to refresh token: %v\n", err)
		} else if time.Now().Before(config.ExpiresAt) {
			fmt.Println("\nUsing cached token!")
			val, err := json.MarshalIndent(config, "", " ")
			if err != nil {
				return "", err
			}
			return string(val), nil
		}
	}

	values := url.Values{}
	values.Add("client_id", clientID)
	values.Add("scope", "read:user user:email repo workflow write:packages read:org") // Expanded GitHub scopes

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
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			Error        string `json:"error"`
		}

		if err := json.Unmarshal(b, &tokenResp); err != nil {
			return "", err
		}

		if tokenResp.AccessToken != "" {
			fmt.Println("\nToken received!")

			// Save token configuration
			config := TokenConfig{
				AccessToken:  tokenResp.AccessToken,
				TokenType:    tokenResp.TokenType,
				RefreshToken: tokenResp.RefreshToken,
				ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
			}

			if err := SaveTokenConfig(config); err != nil {
				fmt.Printf("Warning: Failed to save token configuration: %v\n", err)
			}

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
