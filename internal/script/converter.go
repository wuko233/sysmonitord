package script

import (
	"fmt"
	"os"
	"sysmonitord/internal/event"
	"time"
)

func FromEvent(event *event.Event) ScriptEvent {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	eventTime := event.Time
	if eventTime.IsZero() {
		eventTime = time.Now()
	}

	return ScriptEvent{
		Version:   EventVersion,
		Hostname:  hostname,
		Timestamp: eventTime.Unix(),
		Type:      string(event.Type),
		Source:    event.Source,
		EventID:   buildEventID(eventTime, event),
		Data:      buildEventData(event),
		Context: map[string]any{
			"generator": "sysmonitord",
		},
	}
}

func buildEventID(t time.Time, e *event.Event) string {
	return fmt.Sprintf("evt_%d_%s_%s", t.UnixNano(), e.Type, e.Source)
}

func buildEventData(e *event.Event) map[string]any {
	data := make(map[string]any)
	if e.Path != "" {
		data["path"] = e.Path
	}
	if e.Name != "" {
		data["name"] = e.Name
	}
	if e.PID != 0 {
		data["pid"] = e.PID
	}
	if e.Reason != "" {
		data["reason"] = e.Reason
	}
	if e.Detail != "" {
		data["detail"] = e.Detail
	}
	if e.Data != nil {
		data["raw"] = e.Data
	}
	if e.Err != nil {
		data["error"] = e.Err.Error()
	}
	return data
}
