package config

import (
	"os"
	"strconv"
)

type Config struct {
	DeezerARL           string
	DeezerQuality       int // 0=MP3_128  1=MP3_320  2=FLAC  3=Hi-Res  4=Best
	IncompletePath      string
	CompletePath        string
	CompletePathMapping string
	Port                string
	ScheduleStart       string
	ScheduleEnd         string
}

func Load() Config {
	return Config{
		DeezerARL:           getEnv("DEEZER_ARL", ""),
		DeezerQuality:       getEnvInt("DEEZER_QUALITY", 2), // default: FLAC
		IncompletePath:      getEnv("AUDYN_INCOMPLETE_PATH", "/downloads/incomplete"),
		CompletePath:        getEnv("AUDYN_COMPLETE_PATH", "/downloads/complete"),
		CompletePathMapping: getEnv("AUDYN_COMPLETE_PATH_MAPPING", "/downloads/complete"),
		Port:                getEnv("AUDYN_PORT", "5000"),
		ScheduleStart:       getEnv("AUDYN_SCHEDULE_START", ""),
		ScheduleEnd:         getEnv("AUDYN_SCHEDULE_END", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
