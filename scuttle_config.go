package main

import (
	"fmt"
	"os"
)

type ScuttleConfig struct {
	LoggingEnabled     bool
	EnvoyAdminAPI      string
	StartWithoutEnvoy  bool
	IstioQuitAPI       string
	NeverKillIstio     bool
	IstioFallbackPkill bool
}

func log(message string) {
	if config.LoggingEnabled {
		fmt.Println("scuttle: " + message)
	}
}

func getConfig() ScuttleConfig {
	loggingEnabled := getBoolFromEnv("SCUTTLE_LOGGING", true, false)
	config := ScuttleConfig{
		// Logging enabled by default, disabled if "false"
		LoggingEnabled:     loggingEnabled,
		EnvoyAdminAPI:      getStringFromEnv("ENVOY_ADMIN_API", "", loggingEnabled),
		StartWithoutEnvoy:  getBoolFromEnv("START_WITHOUT_ENVOY", false, loggingEnabled),
		IstioQuitAPI:       getStringFromEnv("ISTIO_QUIT_API", "", loggingEnabled),
		NeverKillIstio:     getBoolFromEnv("NEVER_KILL_ISTIO", false, loggingEnabled),
		IstioFallbackPkill: getBoolFromEnv("ISTIO_FALLBACK_PKILL", false, loggingEnabled),
	}

	return config
}

func getStringFromEnv(name string, defaultVal string, logEnabled bool) string {
	userVal := os.Getenv(name)
	if logEnabled {
		log(fmt.Sprintf("%s: %s", name, userVal))
	}
	if userVal != "" {
		return userVal
	}
	return defaultVal
}

func getBoolFromEnv(name string, defaultVal bool, logEnabled bool) bool {
	userVal := os.Getenv(name)
	// User did not set anything return default
	if userVal == "" {
		return defaultVal
	}

	// User set something, check it is valid
	if userVal != "true" && userVal != "false" {
		if logEnabled {
			log(fmt.Sprintf("%s: %s (Invalid value will be ignored)", name, userVal))
		}
		return defaultVal
	}

	// User gave valid option
	if logEnabled {
		log(fmt.Sprintf("%s: %s", name, userVal))
	}
	return (userVal == "true")
}
