package config

import (
	"github.com/ravan/stackstate-event-emitter/internal/types"
	"github.com/spf13/viper"
	"log/slog"
	"strings"
)

func GetConfig() *types.Configuration {
	c := &types.Configuration{Evt: types.StackStateEvent{}}
	v := viper.New()
	v.SetDefault("api_url", "")
	v.SetDefault("api_key", "")
	v.SetDefault("evt.origin_host", "'localhost'")
	v.SetDefault("evt.source", "emitter")
	v.SetDefault("evt.category", "'Alerts'")
	v.SetDefault("evt.type", "'Emitter Event'")
	v.SetDefault("evt.title", "")
	v.SetDefault("evt.identifier", "")
	v.SetDefault("evt.text", "")
	v.SetDefault("evt.link_title", "")
	v.SetDefault("evt.link_url", "")
	v.SetDefault("evt.tags", "")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.Unmarshal(c); err != nil {
		slog.Error("Error unmarshalling config", slog.Any("err", err))
	}
	return c
}
