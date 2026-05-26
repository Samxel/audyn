package config

import "os"

type Config struct {
	DeezerARL           string
	IncompletePath      string
	CompletePath        string
	CompletePathMapping string
	Port                string
}

func Load() Config {
	return Config{
		DeezerARL:           getEnv("DEEZER_ARL", ""),
		IncompletePath:      getEnv("AUDYN_INCOMPLETE_PATH", "/downloads/incomplete"),
		CompletePath:        getEnv("AUDYN_COMPLETE_PATH", "/downloads/complete"),
		CompletePathMapping: getEnv("AUDYN_COMPLETE_PATH_MAPPING", "/downloads/complete"),
		Port:                getEnv("AUDYN_PORT", "5000"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
