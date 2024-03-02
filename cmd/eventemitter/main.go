package main

import (
	"encoding/json"
	"github.com/ravan/stackstate-event-emitter/internal/config"
	"github.com/ravan/stackstate-event-emitter/internal/sts"
	"log/slog"
	"os"
)

func main() {
	conf := config.GetConfig()
	data := map[string]interface{}{}
	data["body"] = map[string]interface{}{}

	b := os.Getenv("BODY")
	if b == "" {
		slog.Info("Environment variable 'BODY' is not present. Using empty map.")
	} else {
		result := make(map[string]interface{})
		err := json.Unmarshal([]byte(b), &result)
		if err != nil {
			slog.Error("Failed to unmarshall 'BODY' environment variable to json.", slog.Any("error", err))
			os.Exit(1)
		}
		data["body"] = result
	}

	err := sts.SubmitEvent(conf, &data)
	if err != nil {
		slog.Error("Failed to submit event.", slog.Any("error", err))
		os.Exit(1)
	}
}
