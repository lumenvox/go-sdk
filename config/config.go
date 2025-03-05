package config

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"os"
	"reflect"
	"strings"

	"gopkg.in/ini.v1"
)

// ConfigValues holds the SDK application configuration.
type ConfigValues struct {
	AppName          string `ini:"app_name"`
	AppVersion       string `ini:"app_version"`
	LogLevel         string `ini:"log_level"`
	ApiEndpoint      string `ini:"api_endpoint"`
	EnableTls        bool   `ini:"enable_tls"`
	AllowInsecureTls bool   `ini:"allow_insecure_tls"`
	CertificatePath  string `ini:"certificate_path"`
	DeploymentId     string `ini:"deployment_id"`
	OperatorId       string `ini:"operator_id"`
	Username         string `ini:"username"`
	Password         string `ini:"password"`
	IdpEndpoint      string `ini:"idp_endpoint"`
	ClientId         string `ini:"client_id"`
	SecretHash       string `ini:"secret_hash"`
	AuthHeaders      string `ini:"auth_headers"`
	loadedFile       string
	authHeaders      map[string]string
}

// GetConfigValues initializes a new Config instance with default values.
func GetConfigValues(iniFilepath string) (cfg *ConfigValues, err error) {

	cfg = &ConfigValues{
		// Assign default values...
		AppName:          "SDK Example",
		AppVersion:       "1.0.0",
		LogLevel:         "info",
		ApiEndpoint:      "",
		EnableTls:        true,
		AllowInsecureTls: false,
		CertificatePath:  "",
		DeploymentId:     "",
		OperatorId:       "00000000-0000-0000-0000-000000000001",
		Username:         "",
		Password:         "",
		ClientId:         "",
		SecretHash:       "",
		AuthHeaders:      "", // Initialize headers string
		authHeaders:      make(map[string]string),
	}

	err = cfg.Load(iniFilepath)

	return cfg, err
}

// Load initializes the configuration with an optional settings file.
// If `file` is an empty string, no file is loaded. Environment variables may
// be used to override any file-based or default values.
func (configValues *ConfigValues) Load(iniFilepath string) (err error) {

	if iniFilepath != "" {
		// Attempt to load the ini file
		configFromFile, err := ini.Load(iniFilepath)
		if err != nil {
			log.Printf("Failed to load settings from file (%s): %v\n", iniFilepath, err)
		} else {
			configValues.loadedFile = iniFilepath

			// Override defaults with ini file values
			for _, section := range configFromFile.Sections() {
				for key, value := range section.KeysHash() {
					// Correctly convert snake_case to TitleCase
					splitKey := strings.Split(key, "_")
					titleCaseKey := ""
					for _, part := range splitKey {
						part = cases.Title(language.English).String(strings.ToLower(part))
						titleCaseKey += part
					}

					field := reflect.ValueOf(configValues).Elem().FieldByName(titleCaseKey)
					if field.IsValid() {
						if field.Kind() == reflect.Bool {
							field.SetBool(strings.ToLower(value) == "true")
						} else {
							field.SetString(value)
						}
					} else {
						log.Printf("Warning: Configuration key '%s' not found in struct. Ignoring.", key)
					}
				}
			}

			// Load headers from the "headers" key in the INI file.
			configValues.parseHeadersString(configValues.AuthHeaders)
		}
	}

	// Override from environment variables
	envAuthHeaders := os.Getenv("AUTH_HEADERS")
	if envAuthHeaders != "" {
		// Note: clearing out an existing header values here (otherwise they would merge)
		configValues.AuthHeaders = envAuthHeaders
		configValues.authHeaders = make(map[string]string)
		configValues.parseHeadersString(envAuthHeaders)
	}

	// Override from environment variables for other properties (these have LUMENVOX_GO_SDK__ prefixes)
	configEnvVarNames := []string{"APP_NAME", "APP_VERSION", "LOG_LEVEL", "API_ENDPOINT", "ENABLE_TLS",
		"ALLOW_INSECURE_TLS", "CERTIFICATE_PATH", "DEPLOYMENT_ID", "OPERATOR_ID",
		"USERNAME", "PASSWORD", "IDP_ENDPOINT", "CLIENT_ID", "SECRET_HASH"}
	for _, key := range configEnvVarNames {
		envValue := os.Getenv("LUMENVOX_GO_SDK__" + key)
		if envValue != "" {

			// remove the `LUMENVOX_GO_SDK__` prefix before applying
			shortKey := strings.ReplaceAll(envValue, "LUMENVOX_GO_SDK__", "")

			// Correctly convert upper or lowercase snake_case to TitleCase
			splitKey := strings.Split(key, "_")
			titleCaseKey := ""
			for _, part := range splitKey {
				part = cases.Title(language.English).String(strings.ToLower(part))
				titleCaseKey += part
			}

			field := reflect.ValueOf(configValues).Elem().FieldByName(titleCaseKey)
			if field.IsValid() {
				if field.Kind() == reflect.Bool {
					field.SetBool(strings.ToLower(shortKey) == "true")
				} else {
					field.SetString(shortKey)
				}
			} else {
				log.Printf("Warning: Configuration key '%s' not found in struct. Ignoring.", key)
			}
		}
	}

	err = configValues.Validate()
	return err
}

// parseHeadersString parses a comma-delimited string of key=value pairs into the headers map.
func (configValues *ConfigValues) parseHeadersString(headersString string) {

	headerPairs := strings.Split(headersString, ",")
	for _, pair := range headerPairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			log.Printf("Invalid header pair: %s", pair)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		configValues.authHeaders[key] = value
	}
}

// GetAuthHeaders returns the auth headers map
func (configValues *ConfigValues) GetAuthHeaders() map[string]string {

	return configValues.authHeaders
}

// Validate checks if necessary configuration fields are provided and returns
// an error if any required field is missing.
func (configValues *ConfigValues) Validate() (err error) {

	if configValues.ApiEndpoint == "" {
		return fmt.Errorf("api_endpoint is required")
	}

	if configValues.DeploymentId == "" {
		return fmt.Errorf("deployment_id is required")
	}

	if configValues.Username != "" {
		// Auth enabled - validate other fields
		if configValues.IdpEndpoint == "" {
			return fmt.Errorf("idp_endpoint is required")
		}

		if configValues.ClientId == "" {
			return fmt.Errorf("client_id is required")
		}

		if configValues.SecretHash == "" {
			return fmt.Errorf("secret_hash is required")
		}

		if configValues.AuthHeaders == "" {
			return fmt.Errorf("auth_headers is required")
		}

		if configValues.Password == "" {
			return fmt.Errorf("password is required")
		}
	}

	return nil
}
