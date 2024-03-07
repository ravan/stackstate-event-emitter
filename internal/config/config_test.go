package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConf(t *testing.T) {
	os.Setenv("API_URL", "http://localhost")
	os.Setenv("API_KEY", "token")
	os.Setenv("METRIC_NAME", "mymetric")
	os.Setenv("EVT_ORIGIN_HOST", "localhost")
	os.Setenv("EVT_SOURCE", "testsource")
	os.Setenv("EVT_CATEGORY", "Alert")
	os.Setenv("EVT_TYPE", "Test Event")
	os.Setenv("EVT_TITLE", "My Event")
	os.Setenv("EVT_TEXT", "This is a test event")
	os.Setenv("EVT_LINK_TITLE", "MyLink")
	os.Setenv("EVT_LINK_URL", "http://localhost/test")
	os.Setenv("EVT_TAGS", "'test', body.emitter")

	conf := GetConfig()
	assert.Equal(t, conf.ApiUrl, "http://localhost")
	assert.Equal(t, conf.ApiKey, "token")
	assert.Equal(t, conf.MetricName, "mymetric")
	assert.Equal(t, conf.Evt.OriginHost, "localhost")
	assert.Equal(t, conf.Evt.Source, "testsource")
	assert.Equal(t, conf.Evt.Category, "Alert")
	assert.Equal(t, conf.Evt.Type, "Test Event")
	assert.Equal(t, conf.Evt.Title, "My Event")
	assert.Equal(t, conf.Evt.Text, "This is a test event")
	assert.Equal(t, conf.Evt.LinkTitle, "MyLink")
	assert.Equal(t, conf.Evt.LinkUrl, "http://localhost/test")
	assert.Equal(t, conf.Evt.LinkUrl, "http://localhost/test")
	assert.Equal(t, conf.Evt.Tags, []string{"'test'", " body.emitter"})
}
