package auth

import (
	"github.com/lumenvox/go-sdk/logging"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

var enableVerboseLogging = func() bool {
	return os.Getenv("LUMENVOX_GO_SDK__ENABLE_VERBOSE_LOGGING") == "true"
}()

// AuthSettings defines the configuration required for authentication,
// including credentials and headers.
type AuthSettings struct {
	AuthUrl     string // OAuth endpoint URL
	ClientId    string
	SecretHash  string
	AuthHeaders map[string]string
	Username    string
	Password    string
}

// validate checks if all required fields in AuthSettings are set and assigns
// default values for missing fields if applicable.
func (authSettings AuthSettings) validate() (err error) {

	if authSettings.AuthUrl == "" {
		return fmt.Errorf("auth_url is required")
	}

	if authSettings.Username == "" {
		return fmt.Errorf("username is required")
	}

	if authSettings.ClientId == "" {
		return fmt.Errorf("client_id is required")
	}

	if authSettings.SecretHash == "" {
		return fmt.Errorf("secret_hash is required")
	}

	if len(authSettings.AuthHeaders) == 0 {
		// Use defaults if none provided
		authSettings.AuthHeaders = map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSCognitoIdentityProviderService.InitiateAuth",
		}
	}

	if authSettings.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// CognitoProvider uses AWS Cognito to fetch tokens
type CognitoProvider struct {
	Settings AuthSettings
	Client   *http.Client

	mu          sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

// Global cognito provider and initializer controls
var (
	globalCognitoProvider *CognitoProvider
	globalOnce            sync.Once
	tokenRefreshMinimum   = 30 * time.Minute
)

// GetGlobalCognitoProvider initializes a single global instance safely
// and returns the same instance each subsequent call.
func GetGlobalCognitoProvider(settings AuthSettings) *CognitoProvider {

	// Allow 10 seconds for a response from AWS Cognito
	globalOnce.Do(func() {
		globalCognitoProvider = &CognitoProvider{
			Settings: settings,
			Client:   &http.Client{Timeout: 10 * time.Second},
		}
	})

	return globalCognitoProvider
}

// cognitoAuthResponse represents the response from an AWS Cognito
// authentication request.
// It contains authentication results and potential challenge information.
type cognitoAuthResponse struct {
	AuthenticationResult struct {
		AccessToken  string `json:"AccessToken"`
		ExpiresIn    int    `json:"ExpiresIn"`
		TokenType    string `json:"TokenType"`
		IdToken      string `json:"IdToken"`
		RefreshToken string `json:"RefreshToken"`
	} `json:"AuthenticationResult"`
	ChallengeName string `json:"ChallengeName"`
}

// GetToken requests a token from AWS Cognito
func (cognitoProvider *CognitoProvider) GetToken(ctx context.Context) (string, error) {

	cognitoProvider.mu.RLock()
	token := cognitoProvider.cachedToken
	expiry := cognitoProvider.tokenExpiry
	cognitoProvider.mu.RUnlock()

	logger, _ := logging.GetLogger()

	if token != "" && time.Until(expiry) > tokenRefreshMinimum {
		// token expires > tokenRefreshMinimum time from now...
		if enableVerboseLogging {
			logger.Debug("using cached token")
		}

		return token, nil
	}

	// If we reach here, we need to refresh the token
	cognitoProvider.mu.Lock()
	defer cognitoProvider.mu.Unlock()

	// Check once again inside the write lock, another goroutine might have refreshed already
	if cognitoProvider.cachedToken != "" && time.Until(cognitoProvider.tokenExpiry) > tokenRefreshMinimum {
		return cognitoProvider.cachedToken, nil
	}

	validateSettingsErr := cognitoProvider.Settings.validate()
	if validateSettingsErr != nil {
		return "", validateSettingsErr
	}

	payload := map[string]interface{}{
		"AuthParameters": map[string]string{
			"USERNAME":    cognitoProvider.Settings.Username,
			"PASSWORD":    cognitoProvider.Settings.Password,
			"SECRET_HASH": cognitoProvider.Settings.SecretHash,
		},
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": cognitoProvider.Settings.ClientId,
	}

	if enableVerboseLogging {
		logger.Debug("requesting new token")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cognitoProvider.Settings.AuthUrl, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("creating request failed: %w", err)
	}

	// Add required headers (possibly the defaults if none specified)
	for headerName, headerValue := range cognitoProvider.Settings.AuthHeaders {
		req.Header.Set(headerName, headerValue)
	}

	resp, err := cognitoProvider.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s, response: %s", resp.Status, string(bodyBytes))
	}

	var result cognitoAuthResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("decoding json response: %w", err)
	}

	if result.AuthenticationResult.AccessToken == "" {
		return "", fmt.Errorf("authentication succeeded but no AccessToken returned, challenge: %s", result.ChallengeName)
	}

	// Update cached token and its expiry
	cognitoProvider.cachedToken = result.AuthenticationResult.AccessToken
	cognitoProvider.tokenExpiry = time.Now().Add(time.Duration(result.AuthenticationResult.ExpiresIn) * time.Second)

	return cognitoProvider.cachedToken, nil
}
