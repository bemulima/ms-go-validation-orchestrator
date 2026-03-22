package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type HTTPConfig struct {
	Host string
	Port int
}

type EngineEndpoints struct {
	HTML         string
	CSS          string
	React        string
	Node         string
	PHP          string
	PHPFramework string
	NextJS       string
	Browser      string
}

type Config struct {
	ServiceName string
	HTTP        HTTPConfig
	Engines     EngineEndpoints
}

func Load() (Config, error) {
	port, err := intFromEnv("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ServiceName: stringFromEnv("SERVICE_NAME", "ms-go-validation-orchestrator"),
		HTTP: HTTPConfig{
			Host: stringFromEnv("HOST", "0.0.0.0"),
			Port: port,
		},
		Engines: EngineEndpoints{
			HTML:         trimURL(os.Getenv("HTML_VALIDATOR_URL")),
			CSS:          trimURL(os.Getenv("CSS_VALIDATOR_URL")),
			React:        trimURL(os.Getenv("REACT_VALIDATOR_URL")),
			Node:         trimURL(os.Getenv("NODE_VALIDATOR_URL")),
			PHP:          trimURL(os.Getenv("PHP_VALIDATOR_URL")),
			PHPFramework: trimURL(os.Getenv("PHP_FRAMEWORK_VALIDATOR_URL")),
			NextJS:       trimURL(os.Getenv("NEXTJS_VALIDATOR_URL")),
			Browser:      trimURL(os.Getenv("BROWSER_RUNTIME_VALIDATOR_URL")),
		},
	}

	return cfg, nil
}

func stringFromEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func intFromEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return result, nil
}

func trimURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
