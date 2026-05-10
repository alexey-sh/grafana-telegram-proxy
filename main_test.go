package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type Mock struct{}

func (m *Mock) SendTelegramMessage(message string) ([]byte, error) {
	return []byte("hello world"), nil
}

func TestHealthEndpoint(t *testing.T) {
	server := RestController{&Mock{}}
	req, _ := http.NewRequest("GET", "/health", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HealthHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
	assert.Equal(t, "UP", rr.Body.String())
}

func TestSendEndpoint(t *testing.T) {
	server := RestController{&Mock{}}
	body := new(bytes.Buffer)

	payload := Request{
		Title:   "[Alerting] CPU alert",
		Message: "CPU load has reached 65%",
		EvalMatches: []EvalMatch{
			{
				Value:  70.2,
				Metric: "CPU load",
				Tags:   map[string]string{"__name__": "cpu_load"},
			},
		},
		RuleName: "CPU alert",
		RuleURL:  "https://grafana.maslick.ru",
		State:    "alerting",
	}

	_ = json.NewEncoder(body).Encode(payload)
	req, _ := http.NewRequest("POST", "/", body)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.WebhookHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, "hello world", rr.Body.String())
}

func TestFormatMessage_MessageVerbatim(t *testing.T) {
	out := formatMessage(Request{Message: "<b>hi</b> raw"})
	assert.Equal(t, "<b>hi</b> raw", out)
}

func TestFormatMessage_MessageWinsOverAlerts(t *testing.T) {
	out := formatMessage(Request{
		Message: "templated text",
		Alerts:  []Alert{{Status: "firing", Labels: map[string]string{"alertname": "X"}}},
	})
	assert.Equal(t, "templated text", out)
}

func TestFormatMessage_MessageWhitespaceFallsThrough(t *testing.T) {
	out := formatMessage(Request{
		Message: "   \n\t",
		Alerts: []Alert{{
			Status: "firing",
			Labels: map[string]string{"alertname": "Whitespaced"},
		}},
	})
	assert.Contains(t, out, "Whitespaced")
}

func TestFormatMessage_UnifiedFull(t *testing.T) {
	out := formatMessage(Request{
		Status: "firing",
		Title:  "Group title",
		Alerts: []Alert{{
			Status:       "firing",
			Labels:       map[string]string{"alertname": "HighCPU", "instance": "host-1"},
			Annotations:  map[string]string{"summary": "CPU is hot", "description": "host-1 over 90%"},
			Values:       map[string]float64{"A": 91.5},
			DashboardURL: "https://grafana.example/d/abc",
			GeneratorURL: "https://grafana.example/alerting/list",
		}},
	})
	assert.Contains(t, out, "<b>Group title</b>")
	assert.Contains(t, out, "[FIRING]")
	assert.Contains(t, out, "HighCPU")
	assert.Contains(t, out, "CPU is hot")
	assert.Contains(t, out, "host-1 over 90%")
	assert.Contains(t, out, "A=91.5")
	assert.Contains(t, out, "https://grafana.example/d/abc")
	assert.NotContains(t, out, "https://grafana.example/alerting/list")
}

func TestFormatMessage_UnifiedSynthesizedHeader(t *testing.T) {
	out := formatMessage(Request{
		Status: "firing",
		Alerts: []Alert{
			{Status: "firing", Labels: map[string]string{"alertname": "A"}},
			{Status: "firing", Labels: map[string]string{"alertname": "B"}},
			{Status: "firing", Labels: map[string]string{"alertname": "C"}},
		},
	})
	assert.Contains(t, out, "<b>[FIRING:3]</b>")
}

func TestFormatMessage_UnifiedEscapesLabels(t *testing.T) {
	out := formatMessage(Request{
		Alerts: []Alert{{
			Status: "firing",
			Labels: map[string]string{"alertname": "<script>x</script>"},
		}},
	})
	assert.Contains(t, out, "&lt;script&gt;x&lt;/script&gt;")
	assert.NotContains(t, out, "<script>")
}

func TestFormatMessage_LegacyFormat(t *testing.T) {
	out := formatMessage(Request{
		Title:   "[Alerting] CPU alert",
		RuleURL: "https://grafana.example",
		EvalMatches: []EvalMatch{
			{Metric: "CPU load", Value: 70.25},
		},
	})
	assert.Contains(t, out, "<b>[Alerting] CPU alert</b>")
	assert.Contains(t, out, "<i>CPU load : 70.250000</i>")
	assert.Contains(t, out, "https://grafana.example")
}

func TestFormatMessage_LegacyEscapesTitle(t *testing.T) {
	out := formatMessage(Request{
		Title: "<bad>title</bad>",
		EvalMatches: []EvalMatch{
			{Metric: "m<x>", Value: 1.0},
		},
	})
	assert.Contains(t, out, "&lt;bad&gt;title&lt;/bad&gt;")
	assert.Contains(t, out, "m&lt;x&gt;")
	assert.NotContains(t, out, "<bad>")
}

func TestFormatMessage_EmptyPayload(t *testing.T) {
	out := formatMessage(Request{})
	assert.Equal(t, "(empty alert)", out)
}

func TestFormatMessage_TruncatedAlerts(t *testing.T) {
	out := formatMessage(Request{
		Status:          "firing",
		Alerts:          []Alert{{Status: "firing", Labels: map[string]string{"alertname": "A"}}},
		TruncatedAlerts: 4,
	})
	assert.Contains(t, out, "(4 more truncated)")
}

func TestFormatMessage_OverLimitTruncation(t *testing.T) {
	alerts := make([]Alert, 12)
	for i := range alerts {
		alerts[i] = Alert{Status: "firing", Labels: map[string]string{"alertname": "X"}}
	}
	out := formatMessage(Request{Status: "firing", Alerts: alerts})
	assert.Contains(t, out, "(2 more truncated)")
	assert.Equal(t, 10, strings.Count(out, "[FIRING]"))
}

func TestMetricsEndpoint(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	promhttp.Handler().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")
	body := rr.Body.String()
	for _, name := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"telegram_send_total",
		"telegram_send_duration_seconds",
		"service_info",
		"go_goroutines",
		"process_start_time_seconds",
	} {
		assert.Contains(t, body, name)
	}
}

func TestInstrument_RecordsRequestCounter(t *testing.T) {
	before := testutil.ToFloat64(httpRequests.WithLabelValues("test", "GET", "200"))

	noop := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	wrapped := instrument("test", noop)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/anything", nil)
	wrapped(rr, req)

	after := testutil.ToFloat64(httpRequests.WithLabelValues("test", "GET", "200"))
	assert.Equal(t, before+1, after)
}

func TestInstrument_DefaultsTo200WhenWriteHeaderNotCalled(t *testing.T) {
	before := testutil.ToFloat64(httpRequests.WithLabelValues("implicit", "GET", "200"))

	noWriteHeader := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}
	wrapped := instrument("implicit", noWriteHeader)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/anything", nil)
	wrapped(rr, req)

	after := testutil.ToFloat64(httpRequests.WithLabelValues("implicit", "GET", "200"))
	assert.Equal(t, before+1, after)
}
