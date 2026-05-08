package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("grafana-telegram-proxy")

const (
	maxAlertsRendered = 10
	maxAnnotationLen  = 200
)

type RestController struct {
	Service ITelegramSender
}

func (this *RestController) Start() {
	http.HandleFunc("/health", this.HealthHandler)
	if useAuth() {
		http.HandleFunc("/", basicAuth(this.WebhookHandler))
	} else {
		http.HandleFunc("/", this.WebhookHandler)
	}
	fmt.Println("Starting server on port:", strings.Split(getPort(), ":")[1])
	log.Fatal(http.ListenAndServe(getPort(), nil))
}

func useAuth() bool {
	_, usernameOk := os.LookupEnv("USERNAME")
	_, passwordOk := os.LookupEnv("PASSWORD")
	return usernameOk && passwordOk
}

func (_ *RestController) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", 400)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("UP"))
}

func (this *RestController) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", 400)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	log.Info(string(body))

	var message Request
	if err := json.Unmarshal(body, &message); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	resp, err := this.Service.SendTelegramMessage(formatMessage(message))
	if err != nil {
		http.Error(w, "Message delivery failed: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func formatMessage(m Request) string {
	if strings.TrimSpace(m.Message) != "" {
		return m.Message
	}
	if len(m.Alerts) > 0 {
		return formatUnified(m)
	}
	if len(m.EvalMatches) > 0 || m.Title != "" || m.RuleURL != "" {
		return formatLegacy(m)
	}
	return "(empty alert)"
}

func formatUnified(m Request) string {
	var b strings.Builder

	if title := strings.TrimSpace(m.Title); title != "" {
		_, _ = fmt.Fprintf(&b, "<b>%s</b>\n", html.EscapeString(title))
	} else {
		status := strings.ToUpper(m.Status)
		if status == "" {
			status = "ALERT"
		}
		_, _ = fmt.Fprintf(&b, "<b>[%s:%d]</b>\n", html.EscapeString(status), len(m.Alerts))
	}

	if len(m.CommonLabels) > 0 {
		_, _ = b.WriteString(renderLabelsLine(m.CommonLabels))
		_, _ = b.WriteString("\n")
	}

	limit := len(m.Alerts)
	if limit > maxAlertsRendered {
		limit = maxAlertsRendered
	}
	for i := 0; i < limit; i++ {
		writeAlert(&b, m.Alerts[i])
	}

	hidden := len(m.Alerts) - limit + m.TruncatedAlerts
	if hidden > 0 {
		_, _ = fmt.Fprintf(&b, "(%d more truncated)\n", hidden)
	}

	return strings.TrimRight(b.String(), "\n")
}

func writeAlert(b *strings.Builder, a Alert) {
	name := a.Labels["alertname"]
	if name == "" {
		name = firstLabelValue(a.Labels)
	}
	status := strings.ToUpper(a.Status)
	if status == "" {
		status = "ALERT"
	}
	fmt.Fprintf(b, "\n[%s] %s\n", html.EscapeString(status), html.EscapeString(name))

	if s := strings.TrimSpace(a.Annotations["summary"]); s != "" {
		_, _ = fmt.Fprintf(b, "  %s\n", html.EscapeString(truncate(s, maxAnnotationLen)))
	}
	if d := strings.TrimSpace(a.Annotations["description"]); d != "" {
		_, _ = fmt.Fprintf(b, "  %s\n", html.EscapeString(truncate(d, maxAnnotationLen)))
	}

	if len(a.Values) > 0 {
		_, _ = fmt.Fprintf(b, "  %s\n", html.EscapeString(renderValues(a.Values)))
	}

	if link := primaryLink(a); link != "" {
		_, _ = fmt.Fprintf(b, "  %s\n", link)
	}
}

func formatLegacy(m Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<b>%s</b>\n", html.EscapeString(m.Title))
	for _, v := range m.EvalMatches {
		_, _ = fmt.Fprintf(&b, "<i>%s : %f</i>\n", html.EscapeString(v.Metric), v.Value)
	}
	_, _ = b.WriteString(html.EscapeString(m.RuleURL))
	return strings.TrimRight(b.String(), "\n")
}

func primaryLink(a Alert) string {
	for _, u := range []string{a.DashboardURL, a.PanelURL, a.GeneratorURL} {
		if u != "" {
			return u
		}
	}
	return ""
}

func renderLabelsLine(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", html.EscapeString(k), html.EscapeString(labels[k])))
	}
	return strings.Join(parts, " ")
}

func renderValues(values map[string]float64) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%g", k, values[k]))
	}
	return strings.Join(parts, " ")
}

func firstLabelValue(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return labels[keys[0]]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
