package types

type Configuration struct {
	ApiUrl string          `mapstructure:"api_url"`
	ApiKey string          `mapstructure:"api_key"`
	Evt    StackStateEvent `mapstructure:"evt"`
	MetricName string       `mapstructure:"metric_name"`
}

type StackStateEvent struct {
	OriginHost string `mapstructure:"origin_host"`
	Source     string
	Category   string
	Type       string
	Title      string
	Text       string
	Identifier string
	LinkTitle  string `mapstructure:"link_title"`
	LinkUrl    string `mapstructure:"link_url"`
	Tags       []string
}
