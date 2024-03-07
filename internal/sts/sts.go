package sts

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	celext "github.com/google/cel-go/ext"
	"github.com/ravan/stackstate-event-emitter/internal/types"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
)

const (
	// StackStateEndpoint is the path of StackState's event API
	StackStateEndpoint       string = "receiver/stsAgent/intake"
	StackStateMetricEndpoint string = "receiver/stsAgent/api/v1/series"
)

func SubmitEvent(conf *types.Configuration, data *map[string]interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	env, err := makeCelEnv()
	if err != nil {
		return err
	}

	var payload *stackstatePayload
	payload = mustBindToEventPayload(env, data, &conf.Evt)

	err = sendEvent(conf.ApiUrl, conf.ApiKey, payload)
	if err != nil {
		return err
	}

	metric := mustEvalString(conf.MetricName, env, data)
	if metric != "" {
		sendMetric(conf.ApiUrl, conf.ApiKey, bindToMetricSeries(metric, payload))
	}

	return nil
}

func sendEvent(apiUrl string, apiKey string, payload *stackstatePayload) error {
	apiUrl, _ = strings.CutSuffix(apiUrl, "/")
	agentEndpoint := fmt.Sprintf("%s/%s?api_key=%s", apiUrl, StackStateEndpoint, apiKey)

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := client.Post(agentEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Unexpected error.", slog.Any("error", err))
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		slog.Error("Failed to post payload.", slog.String("payload", string(body)),
			slog.Int("status", resp.StatusCode),
			slog.String("response", string(b)))
		return fmt.Errorf("Failed to send event. Status '%v'. Response: %s", resp.StatusCode, string(b))
	} else {
		slog.Info("Sent event.", slog.String("event", string(body)))
	}

	return nil
}

func sendMetric(apiUrl string, apiKey string, payload *MetricSeries) {
	apiUrl, _ = strings.CutSuffix(apiUrl, "/")
	agentEndpoint := fmt.Sprintf("%s/%s?api_key=%s", apiUrl, StackStateMetricEndpoint, apiKey)

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal payload. Metric not sent.", slog.Any("error", err))
		return
	}
	resp, err := client.Post(agentEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("Failed to post payload. Metric not sent.", slog.Any("error", err))
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Unexpected error.", slog.Any("error", err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		slog.Error("Failed to post metric payload.", slog.String("payload", string(body)),
			slog.Int("status", resp.StatusCode),
			slog.String("response", string(b)))
		return
	} else {
		slog.Info("Sent metric.", slog.String("metric", string(body)))
	}
}

func bindToMetricSeries(name string, payload *stackstatePayload) *MetricSeries {
	var tags []string
	ep := payload.Events["emitter_event"][0]
	for _, t := range ep.Tags {
		if strings.Contains(t, ":") {
			tags = append(tags, t)
		}
	}
	tags = append(tags, fmt.Sprintf("event_type:%s", ep.EventType))
	identifier := ep.Context.ElementIdentifiers[0]
	tags = append(tags, fmt.Sprintf("identifier:%s", identifier))

	ms := MetricSeries{
		Series: []Metric{
			{
				Name: name,
				Points: []Point{
					{
						Timestamp: payload.CollectionTimestamp,
						Value:     1,
					},
				},
				Tags:           tags,
				Host:           identifier,
				Type:           "gauge",
				Interval:       0,
				SourceTypeName: ep.SourceTypeName,
			},
		},
	}
	return &ms
}

func mustBindToEventPayload(env *cel.Env, data *map[string]interface{}, evt *types.StackStateEvent) *stackstatePayload {
	payload := emptyPayload()
	payload.CollectionTimestamp = time.Now().Unix()
	payload.InternalHostname = mustEvalString(evt.OriginHost, env, data)
	payload.InternalHostname = mustEvalString(evt.OriginHost, env, data)
	ep := eventPayload{}
	ep.Context = eventContext{}
	ep.Timestamp = time.Now().Unix()
	ep.EventType = mustEvalString(evt.Type, env, data)
	ep.Title = mustEvalString(evt.Title, env, data)
	ep.Text = mustEvalString(evt.Text, env, data)
	ep.SourceTypeName = evt.Source

	ep.Tags = evt.Tags
	if len(evt.Tags) > 0 {
		ep.Tags = []string{}
		for _, tag := range evt.Tags {
			ep.Tags = append(ep.Tags, mustEvalString(tag, env, data))
		}
	}

	ep.Context.Category = mustEvalString(evt.Category, env, data)
	ep.Context.Source = evt.Source
	identifier := mustEvalString(evt.Identifier, env, data)
	if identifier != "" {
		ep.Context.ElementIdentifiers = []string{identifier}
	}

	ep.Context.SourceLinks = []eventLink{}
	linkTitle := mustEvalString(evt.LinkTitle, env, data)
	linkUrl := mustEvalString(evt.LinkUrl, env, data)
	if linkUrl != "" && linkTitle != "" {
		sl := eventLink{
			Title: linkTitle,
			URL:   linkUrl,
		}
		ep.Context.SourceLinks = []eventLink{sl}

	}
	payload.Events["emitter_event"] = []eventPayload{ep}
	return payload
}

func mustEvalString(s string, env *cel.Env, data *map[string]interface{}) string {
	s = strings.TrimSpace(s)
	if s == "''" || s == "" {
		return ""
	}
	out, err := evaluate(s, env, data)
	if err != nil {
		slog.Error("Failed to evaluate cel expression", slog.String("expr", s))
		panic(err)
	}
	if stringValue, ok := out.Value().(string); ok {
		return stringValue
	} else {
		slog.Error("Failed to evaluate cel expression as string", slog.String("expr", s), slog.Any("val", out.Value()))
		panic(fmt.Errorf("Failed to evaluate cel expression as string. %s", s))
	}
}

func makeCelEnv() (*cel.Env, error) {
	mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
	return cel.NewEnv(
		celext.Strings(),
		cel.Declarations(
			decls.NewVar("body", mapStrDyn),
		))
}

func evaluate(expr string, env *cel.Env, data *map[string]interface{}) (ref.Val, error) {
	parsed, issues := env.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to parse expression %#v: %w", expr, issues.Err())
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("expression %#v check failed: %w", expr, issues.Err())
	}

	prg, err := env.Program(checked, cel.EvalOptions(cel.OptOptimize))
	if err != nil {
		return nil, fmt.Errorf("expression %#v failed to create a Program: %w", expr, err)
	}

	out, _, err := prg.Eval(*data)
	if err != nil {
		return nil, fmt.Errorf("expression %#v failed to evaluate: %w", expr, err)
	}
	return out, nil
}

type stackstatePayload struct {
	CollectionTimestamp int64           `json:"collection_timestamp"` // Epoch timestamp in seconds
	InternalHostname    string          `json:"internalHostname"`     // The hostname sending the data
	Events              events          `json:"events"`               // The events to send to StackState
	Metrics             []metrics       `json:"metrics"`              // Required present, but can be empty
	ServiceChecks       []serviceChecks `json:"service_checks"`       // Required present, but can be empty
	Health              []health        `json:"health"`               // Required present, but can be empty
	Topologies          []topology      `json:"topologies"`           // Required present, but can be empty
}

type events map[string][]eventPayload

type eventPayload struct {
	Context        eventContext `json:"context"`
	EventType      string       `json:"event_type"`
	Title          string       `json:"msg_title"`
	Text           string       `json:"msg_text"`
	SourceTypeName string       `json:"source_type_name"`
	Tags           []string     `json:"tags"`
	Timestamp      int64        `json:"timestamp"`
}

type eventContext struct {
	Category           string            `json:"category"`            // The event category. Can be Activities, Alerts, Anomalies, Changes or Others.
	Data               map[string]string `json:"data"`                // Optional. A list of key/value details about the event, for example a configuration version.
	ElementIdentifiers []string          `json:"element_identifiers"` // The identifiers for the topology element(s) the event relates to. These are used to bind the event to a topology element or elements.
	Source             string            `json:"source"`              // The name of the system from which the event originates, for example AWS, Kubernetes or JIRA.
	SourceLinks        []eventLink       `json:"source_links"`
}

type eventLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type serviceChecks struct{}
type health struct{}
type metrics struct{}
type topology struct{}

func emptyPayload() *stackstatePayload {
	return &stackstatePayload{
		Events:        events{},
		Metrics:       []metrics{},
		ServiceChecks: []serviceChecks{},
		Health:        []health{},
		Topologies:    []topology{},
	}
}

type MetricSeries struct {
	Series []Metric `json:"series"`
}

type Metric struct {
	Name           string   `json:"metric"`
	Points         []Point  `json:"points"`
	Tags           []string `json:"tags"`
	Host           string   `json:"host"`
	Type           string   `json:"type"`
	Interval       int      `json:"interval"`
	SourceTypeName string   `json:"source_type_name"`
}

type Point struct {
	Timestamp int64
	Value     float32
}

func (t *Point) MarshalJSON() ([]byte, error) {
	return json.Marshal(&[]interface{}{
		t.Timestamp,
		t.Value,
	})
}
