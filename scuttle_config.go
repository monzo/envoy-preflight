package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type ScuttleConfig struct {
	LoggingEnabled          bool
	EnvoyAdminAPI           string
	StartWithoutEnvoy       bool
	WaitForEnvoyTimeout     time.Duration
	IstioQuitAPI            string
	NeverKillIstio          bool
	IstioFallbackPkill      bool
	NeverKillIstioOnFailure bool
	GenericQuitEndpoints    []string
}

func log(message string) {
	if config.LoggingEnabled {
		fmt.Printf("%s scuttle: %s\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"), message)
	}
}

func getConfig() ScuttleConfig {
	loggingEnabled := getBoolFromEnv("SCUTTLE_LOGGING", true, false)
	config := ScuttleConfig{
		// Logging enabled by default, disabled if "false"
		LoggingEnabled:          loggingEnabled,
		EnvoyAdminAPI:           getStringFromEnv("ENVOY_ADMIN_API", "", loggingEnabled),
		StartWithoutEnvoy:       getBoolFromEnv("START_WITHOUT_ENVOY", false, loggingEnabled),
		WaitForEnvoyTimeout:     getDurationFromEnv("WAIT_FOR_ENVOY_TIMEOUT", time.Duration(0), loggingEnabled),
		IstioQuitAPI:            getStringFromEnv("ISTIO_QUIT_API", "", loggingEnabled),
		NeverKillIstio:          getBoolFromEnv("NEVER_KILL_ISTIO", false, loggingEnabled),
		IstioFallbackPkill:      getBoolFromEnv("ISTIO_FALLBACK_PKILL", false, loggingEnabled),
		NeverKillIstioOnFailure: getBoolFromEnv("NEVER_KILL_ISTIO_ON_FAILURE", false, loggingEnabled),
		GenericQuitEndpoints:    getStringArrayFromEnv("GENERIC_QUIT_ENDPOINTS", make([]string, 0), loggingEnabled),
	}

	return config
}

func getStringArrayFromEnv(name string, defaultVal []string, logEnabled bool) []string {
	userValCsv := strings.Trim(os.Getenv(name), " ")

	if userValCsv == "" {
		return defaultVal
	}

	if logEnabled {
		log(fmt.Sprintf("%s: %s", name, userValCsv))
	}

	userValArray := strings.Split(userValCsv, ",")
	if len(userValArray) == 0 {
		return defaultVal
	}

	return userValArray
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
	return userVal == "true"
}

func getDurationFromEnv(name string, defaultVal time.Duration, logEnabled bool) time.Duration {
	userVal := os.Getenv(name)

	// User did not set anything, return default.
	if userVal == "" {
		return defaultVal
	}

	// User has set something, check it is valid.
	if userVal != "" {
		if duration, err := time.ParseDuration(userVal); err == nil {
			// User gave valid option.
			if logEnabled {
				log(fmt.Sprintf("%s: %s", name, userVal))
			}

			return duration
		} else if logEnabled {
			log(fmt.Sprintf("%s: %s (Invalid value will be ignored)", name, userVal))
		}
	}

	return defaultVal
}
