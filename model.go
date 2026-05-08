package main

import "time"

type Request struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	OrgID             int64             `json:"orgId"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	State             string            `json:"state"`
	Title             string            `json:"title"`
	Message           string            `json:"message"`

	EvalMatches []EvalMatch `json:"evalMatches"`
	RuleName    string      `json:"ruleName"`
	RuleURL     string      `json:"ruleUrl"`
}

type Alert struct {
	Status       string             `json:"status"`
	Labels       map[string]string  `json:"labels"`
	Annotations  map[string]string  `json:"annotations"`
	StartsAt     time.Time          `json:"startsAt"`
	EndsAt       time.Time          `json:"endsAt"`
	GeneratorURL string             `json:"generatorURL"`
	Fingerprint  string             `json:"fingerprint"`
	SilenceURL   string             `json:"silenceURL"`
	DashboardURL string             `json:"dashboardURL"`
	PanelURL     string             `json:"panelURL"`
	ImageURL     string             `json:"imageURL"`
	Values       map[string]float64 `json:"values"`
}

type EvalMatch struct {
	Value  float64           `json:"value"`
	Metric string            `json:"metric"`
	Tags   map[string]string `json:"tags"`
}

type Message struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}
