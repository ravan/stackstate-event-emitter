package sts

import (
	"fmt"
	"github.com/h2non/gock"
	"github.com/ravan/stackstate-event-emitter/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBindEvent(t *testing.T) {
	conf := getConf()
	conf.Evt.OriginHost = "string(body.int) + 'localhost'"

	env, err := makeCelEnv()
	require.NoError(t, err)

	data := map[string]interface{}{"body": map[string]interface{}{
		"string": "myString",
		"int":    1,
	}}

	payload := mustBindToEventPayload(env, &data, &conf.Evt)
	assert.Equal(t, "1localhost", payload.InternalHostname)
	assert.True(t, payload.CollectionTimestamp > 0)
	assert.Equal(t, 1, len(payload.Events))
	e := payload.Events["emitter_event"][0]
	assert.Equal(t, "Test Event", e.EventType)
	assert.Equal(t, "A test event", e.Text)
	assert.Equal(t, "My Event", e.Title)
	assert.Equal(t, "testsource", e.SourceTypeName)
	assert.Equal(t, "Alert", e.Context.Category)
	assert.Equal(t, "urn:host:test:/myString", e.Context.ElementIdentifiers[0])
	assert.Equal(t, "mytitle", e.Context.SourceLinks[0].Title)
	assert.Equal(t, "http://mylink", e.Context.SourceLinks[0].URL)
	assert.Equal(t, "test", e.Tags[0])
}

func TestSubmitEvent(t *testing.T) {
	conf := getConf()
	defer gock.Off() // Flush pending mocks after test execution
	gock.Observe(gock.DumpRequest)
	gock.New(conf.ApiUrl).
		Post(fmt.Sprintf("/%s", StackStateEndpoint)).
		MatchParam("api_key", conf.ApiKey).
		Reply(200)
	gock.New(conf.ApiUrl).
		Post(fmt.Sprintf("/%s", StackStateMetricEndpoint)).
		MatchParam("api_key", conf.ApiKey).
		Reply(200)
	gock.InterceptClient(client)
	data := map[string]interface{}{"body": map[string]interface{}{
		"string": "myString",
	}}

	err := SubmitEvent(&conf, &data)
	require.NoError(t, err)
}

func getConf() types.Configuration {
	return types.Configuration{
		ApiKey: "test",
		ApiUrl: "http://localhost",
		Evt: types.StackStateEvent{
			OriginHost: "'localhost'",
			Source:     "testsource",
			Category:   "'Alert'",
			Type:       "'Test Event'",
			Title:      "'My Event'",
			Text:       "'A test event'",
			Identifier: "'urn:host:test:/' + body.string",
			LinkTitle:  "'mytitle'",
			LinkUrl:    "'http://mylink'",
			Tags:       []string{"'test'"},
		},
	}
}
