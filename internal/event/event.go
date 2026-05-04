package event

import "time"

type Type string

const (
	TypeSystemStart    Type = "system_start"
	TypeSystemStop     Type = "system_stop"
	TypeFileChange     Type = "file_change"
	TypeDubiousFile    Type = "dubious_file"
	TypeDubiousProcess Type = "dubious_process"
	TypeError          Type = "error"
)

type Event struct {
	Time   time.Time `json:"time"`
	Type   Type      `json:"type"`
	Source string    `json:"source"`
	Path   string    `json:"path,omitempty"`
	Name   string    `json:"name,omitempty"`
	PID    int32     `json:"pid,omitempty"`
	Reason string    `json:"reason,omitempty"`
	Detail string    `json:"detail,omitempty"`
	Data   any       `json:"data,omitempty"`
	Err    error     `json:"-"`
}

type Handler interface {
	Handle(e Event)
}

type FuncHandler func(e Event)

func (f FuncHandler) Handle(e Event) {
	f(e)
}
