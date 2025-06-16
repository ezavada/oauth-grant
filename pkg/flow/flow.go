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

func OauthFlow(clientID string, verbose bool) (string, error) {
	return GetGitHubDeviceFlowToken(clientID, verbose)
}

func refreshToken(clientID string, refreshToken string, verbose bool) (*TokenConfig, error) {
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
		if verbose {
			fmt.Printf("Warning: Failed to save refreshed token configuration: %v\n", err)
		}
	}

	return config, nil
}

func GetGitHubDeviceFlowToken(clientID string, verbose bool) (string, error) {
	// First check if we have a valid token in the config
	if config, err := LoadTokenConfig(); err == nil && config != nil {
		// If token is expired but we have a refresh token, try to refresh it
		if time.Now().After(config.ExpiresAt) && config.RefreshToken != "" {
			if verbose {
				fmt.Println("\nToken expired, attempting to refresh...")
			}
			newConfig, err := refreshToken(clientID, config.RefreshToken, verbose)
			if err == nil && newConfig != nil {
				if verbose {
					fmt.Println("\nToken refreshed successfully!")
				}
				val, err := json.MarshalIndent(newConfig, "", " ")
				if err != nil {
					return "", err
				}
				return string(val), nil
			}
			if verbose {
				fmt.Printf("\nFailed to refresh token: %v\n", err)
			}
		} else if time.Now().Before(config.ExpiresAt) {
			if verbose {
				fmt.Println("\nUsing cached token!")
			}
			val, err := json.MarshalIndent(config, "", " ")
			if err != nil {
				return "", err
			}
			return string(val), nil
		}
	}

	// Create JSON request body
	jsonBody := map[string]string{
		"client_id": clientID,
	}
	jsonData, err := json.Marshal(jsonBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", githubDeviceAuthEndpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Print the request body for debugging
	if verbose {
		fmt.Printf("\nRequest body: %s\n", string(jsonData))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if verbose {
		fmt.Printf("\nRaw device auth response: %s\n", string(b))
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %s ", resp.Status)
	}

	var dr DeviceResp
	if err := json.Unmarshal(b, &dr); err != nil {
		return "", fmt.Errorf("error while unmarshaling device response: %s", err)
	}

	uri := dr.VerificationURI

	fmt.Printf("\nOpen link : %s in browser and enter verification code %s\n", uri, dr.UserCode)
	fmt.Printf("\nCode will be valid for %d seconds\n", dr.ExpiresIn)

	for {
		// Create JSON request body for token request
		tokenJsonBody := map[string]string{
			"client_id":   clientID,
			"device_code": dr.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}
		tokenJsonData, err := json.Marshal(tokenJsonBody)
		if err != nil {
			return "", err
		}

		req, err := http.NewRequest("POST", githubTokenEndpoint, strings.NewReader(string(tokenJsonData)))
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if verbose {
			fmt.Printf("\nRaw token response: %s\n", string(b))
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
			if verbose {
				fmt.Println("\nToken received!")
			}

			// Save token configuration
			config := TokenConfig{
				AccessToken:  tokenResp.AccessToken,
				TokenType:    tokenResp.TokenType,
				RefreshToken: tokenResp.RefreshToken,
				ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
			}

			if err := SaveTokenConfig(config); err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to save token configuration: %v\n", err)
				}
			}

			val, err := json.MarshalIndent(tokenResp, "", " ")
			if err != nil {
				return "", err
			}
			return string(val), nil
		}

		switch tokenResp.Error {
		case "authorization_pending":
			if verbose {
				fmt.Printf("\nAuthorization request is still pending. Waiting for %d seconds...\n", dr.Interval)
			}
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

func GetAccessToken(clientID string) (string, error) {
	tokenResp, err := OauthFlow(clientID, false)
	if err != nil {
		return "", err
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tokenResp), &token); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if token.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return token.AccessToken, nil
}
