package script

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dop251/goja"
)

type Engine struct {
	Timeout time.Duration
}

func NewEngine(timeoutMS int) *Engine {
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}
	return &Engine{
		Timeout: time.Duration(timeoutMS) * time.Millisecond,
	}
}

func (e *Engine) ExecuteFile(scriptPath string, event ScriptEvent) (ScriptResult, error) {
	code, err := os.ReadFile(scriptPath)
	if err != nil {
		return ScriptResult{}, fmt.Errorf("read script file failed: %w", err)
	}
	return e.Execute(string(code), event)
}

func (e *Engine) Execute(code string, event ScriptEvent) (ScriptResult, error) {
	vm := goja.New()
	eventValue, err := structToJSValue(vm, event)
	if err != nil {
		return ScriptResult{}, err
	}
	if err := vm.Set("event", eventValue); err != nil {
		return ScriptResult{}, fmt.Errorf("set event failed: %w", err)
	}
	if _, err := vm.RunString(code); err != nil {
		return ScriptResult{}, fmt.Errorf("run script failed: %w", err)
	}
	handleValue := vm.Get("handle")
	if goja.IsUndefined(handleValue) || goja.IsNull(handleValue) {
		return ScriptResult{}, fmt.Errorf("script must define handle(event) function")
	}
	handleFunc, ok := goja.AssertFunction(handleValue)
	if !ok {
		return ScriptResult{}, fmt.Errorf("handle is not a function")
	}
	resultValue, err := e.callWithTimeout(vm, handleFunc, eventValue)
	if err != nil {
		return ScriptResult{}, err
	}
	var result ScriptResult
	if err := jsValueToStruct(resultValue, &result); err != nil {
		return ScriptResult{}, err
	}
	if result.Action == "" {
		result.Action = "log"
	}
	if result.Level == "" {
		result.Level = "info"
	}
	return result, nil
}

func (e *Engine) callWithTimeout(vm *goja.Runtime, fn goja.Callable, eventValue goja.Value) (goja.Value, error) {
	done := make(chan struct{})
	var result goja.Value
	var callErr error
	timer := time.AfterFunc(e.Timeout, func() {
		vm.Interrupt("script execution timeout")
	})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("script execution interrupted: %v", r)
			}
		}()
		result, callErr = fn(goja.Undefined(), eventValue)
	}()
	<-done
	timer.Stop()
	if callErr != nil {
		return nil, fmt.Errorf("call handle failed: %w", callErr)
	}
	return result, nil
}

func structToJSValue(vm *goja.Runtime, v any) (goja.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal value failed: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal value failed: %w", err)
	}
	return vm.ToValue(m), nil
}

func jsValueToStruct(value goja.Value, out any) error {
	exported := value.Export()
	b, err := json.Marshal(exported)
	if err != nil {
		return fmt.Errorf("marshal js result failed: %w", err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("unmarshal js result failed: %w", err)
	}
	return nil
}
