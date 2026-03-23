package adapters

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
)

type ob11EchoWaiter struct {
	waiters          sync.Map // map[string]chan ob11APIResponse
	droppedEchoCount atomic.Uint64
}

func (w *ob11EchoWaiter) register(echo string) chan ob11APIResponse {
	ch := make(chan ob11APIResponse, 1)
	w.waiters.Store(strings.TrimSpace(echo), ch)
	return ch
}

func (w *ob11EchoWaiter) cancel(echo string) {
	w.waiters.Delete(strings.TrimSpace(echo))
}

func (w *ob11EchoWaiter) resolve(rawEcho json.RawMessage, payload []byte) bool {
	echo := sanitizeRawMessage(rawEcho)
	if echo == "" {
		w.droppedEchoCount.Add(1)
		return false
	}

	chAny, ok := w.waiters.LoadAndDelete(echo)
	if !ok {
		w.droppedEchoCount.Add(1)
		return false
	}

	ch, ok := chAny.(chan ob11APIResponse)
	if !ok {
		w.droppedEchoCount.Add(1)
		return false
	}

	var resp ob11APIResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		resp = ob11APIResponse{Status: "failed", Message: err.Error(), Echo: rawEcho}
	}

	select {
	case ch <- resp:
		return true
	default:
		w.droppedEchoCount.Add(1)
		return false
	}
}

func (w *ob11EchoWaiter) failAll(err error) {
	w.waiters.Range(func(key, value any) bool {
		ch, ok := value.(chan ob11APIResponse)
		if ok {
			resp := ob11APIResponse{Status: "failed", Message: err.Error()}
			select {
			case ch <- resp:
			default:
				w.droppedEchoCount.Add(1)
			}
		}
		w.waiters.Delete(key)
		return true
	})
}

func (w *ob11EchoWaiter) pendingKeys() []string {
	var keys []string
	w.waiters.Range(func(key, _ any) bool {
		if k, ok := key.(string); ok {
			keys = append(keys, k)
		}
		return true
	})
	return keys
}

func (w *ob11EchoWaiter) droppedCount() uint64 {
	return w.droppedEchoCount.Load()
}
