package script

import (
	"path/filepath"
	"sysmonitord/internal/config"
	"time"
)

type Manager struct {
	enabled bool
	dir     string
	engine  *Engine
	events  map[string][]string
}

func NewManager(cfg config.ScriptConfig) *Manager {
	events := make(map[string][]string, len(cfg.Events))
	for eventType, scripts := range cfg.Events {
		copied := make([]string, len(scripts))
		copy(copied, scripts)
		events[eventType] = copied
	}
	return &Manager{
		enabled: cfg.Enabled,
		dir:     cfg.Dir,
		engine:  NewEngine(cfg.TimeoutMS),
		events:  events,
	}
}

func (m *Manager) Enabled() bool {
	if m == nil {
		return false
	}
	return m.enabled
}

func (m *Manager) ScriptsForEvent(eventType string) []string {
	if m == nil {
		return nil
	}

	scripts := m.events[eventType]
	copied := make([]string, len(scripts))
	copy(copied, scripts)
	return copied
}

func (m *Manager) ExecuteEvent(event ScriptEvent) []ScriptExecution {
	if m == nil || !m.enabled {
		return nil
	}

	scripts := m.ScriptsForEvent(event.Type)
	if len(scripts) == 0 {
		return nil
	}

	executions := make([]ScriptExecution, 0, len(scripts))
	for _, scriptName := range scripts {
		startTime := time.Now()
		execution := ScriptExecution{
			Time:      startTime.Unix(),
			Script:    scriptName,
			EventID:   event.EventID,
			EventType: event.Type,
		}

		scriptPath := m.resolveScriptPath(scriptName)
		result, err := m.engine.ExecuteFile(scriptPath, event)
		execution.CostMS = time.Since(startTime).Milliseconds()
		if err != nil {
			execution.Success = false
			execution.Error = err.Error()
		} else {
			execution.Success = true
			execution.Result = result
		}
		executions = append(executions, execution)
	}
	return executions
}

func (m *Manager) resolveScriptPath(scriptName string) string {
	if filepath.IsAbs(scriptName) {
		return scriptName
	}
	return filepath.Join(m.dir, scriptName)
}
