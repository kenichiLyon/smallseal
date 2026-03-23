package adapters

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sealdice/smallseal/dice/types"
)

func TestOB11EventDispatcherSubmit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	processed := make(chan string, 2)
	dispatcher := newOB11EventDispatcher(ctx, 1, 2, func(job ob11DispatchJob) {
		processed <- job.postType + ":" + string(job.payload)
	})
	defer func() {
		cancel()
		dispatcher.wait()
	}()

	if err := dispatcher.submit(ctx, ob11DispatchJob{postType: "message", payload: []byte("one")}); err != nil {
		t.Fatalf("submit first job failed: %v", err)
	}
	if err := dispatcher.submit(ctx, ob11DispatchJob{postType: "notice", payload: []byte("two")}); err != nil {
		t.Fatalf("submit second job failed: %v", err)
	}

	for _, want := range []string{"message:one", "notice:two"} {
		select {
		case got := <-processed:
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
}

func TestOB11EventDispatcherSubmitAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := newOB11EventDispatcher(ctx, 1, 1, func(job ob11DispatchJob) {})
	cancel()
	defer dispatcher.wait()

	err := dispatcher.submit(context.Background(), ob11DispatchJob{postType: "message", payload: []byte("x")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestOB11EchoWaiterResolve(t *testing.T) {
	var waiter ob11EchoWaiter
	ch := waiter.register("seal-echo-1")
	resolved := waiter.resolve([]byte(`"seal-echo-1"`), []byte(`{"status":"ok","retcode":0,"data":{"message_id":123},"echo":"seal-echo-1"}`))
	if !resolved {
		t.Fatal("expected echo to resolve")
	}

	select {
	case resp := <-ch:
		if resp.Status != "ok" {
			t.Fatalf("unexpected status: %+v", resp)
		}
		if string(resp.Data) != `{"message_id":123}` {
			t.Fatalf("unexpected data: %s", string(resp.Data))
		}
	default:
		t.Fatal("expected waiter response")
	}

	if len(waiter.pendingKeys()) != 0 {
		t.Fatalf("expected no pending waiters, got %+v", waiter.pendingKeys())
	}
}

func TestOB11EchoWaiterFailAll(t *testing.T) {
	var waiter ob11EchoWaiter
	ch1 := waiter.register("seal-echo-1")
	ch2 := waiter.register("seal-echo-2")
	boom := errors.New("boom")

	waiter.failAll(boom)

	for idx, ch := range []chan ob11APIResponse{ch1, ch2} {
		select {
		case resp := <-ch:
			if resp.Status != "failed" || !strings.Contains(resp.Message, "boom") {
				t.Fatalf("unexpected failed response #%d: %+v", idx, resp)
			}
		default:
			t.Fatalf("expected failed response #%d", idx)
		}
	}

	if len(waiter.pendingKeys()) != 0 {
		t.Fatalf("expected no pending waiters, got %+v", waiter.pendingKeys())
	}
}

func TestOB11FromSegmentsUsesPureOBFallbacks(t *testing.T) {
	pa := &PlatformAdapterOB11{}
	segments := pa.fromSegments([]ob11Segment{
		{Type: "image", Data: map[string]any{"file": "cat.png", "url": "https://img.example/cat.png"}},
		{Type: "record", Data: map[string]any{"file": "voice.amr", "path": "C:/tmp/voice.amr"}},
		{Type: "file", Data: map[string]any{"file": "report.txt", "path": "C:/tmp/report.txt"}},
		{Type: "at", Data: map[string]any{"qq": float64(123456)}},
		{Type: "face", Data: map[string]any{"id": float64(14)}},
		{Type: "reply", Data: map[string]any{"id": float64(9)}},
		{Type: "poke", Data: map[string]any{"qq": float64(10001)}},
		{Type: "mface", Data: map[string]any{"id": float64(1), "name": "test"}},
	})

	if len(segments) != 8 {
		t.Fatalf("expected 8 segments, got %d", len(segments))
	}

	img, ok := segments[0].(*types.ImageElement)
	if !ok || img.URL != "https://img.example/cat.png" {
		t.Fatalf("unexpected image segment: %#v", segments[0])
	}

	record, ok := segments[1].(*types.RecordElement)
	if !ok || record.File == nil || record.File.URL != "C:/tmp/voice.amr" {
		t.Fatalf("unexpected record segment: %#v", segments[1])
	}

	file, ok := segments[2].(*types.FileElement)
	if !ok || file.URL != "C:/tmp/report.txt" {
		t.Fatalf("unexpected file segment: %#v", segments[2])
	}

	at, ok := segments[3].(*types.AtElement)
	if !ok || at.Target != "123456" {
		t.Fatalf("unexpected at segment: %#v", segments[3])
	}

	face, ok := segments[4].(*types.FaceElement)
	if !ok || face.FaceID != "14" {
		t.Fatalf("unexpected face segment: %#v", segments[4])
	}

	reply, ok := segments[5].(*types.ReplyElement)
	if !ok || reply.ReplySeq != "9" {
		t.Fatalf("unexpected reply segment: %#v", segments[5])
	}

	poke, ok := segments[6].(*types.PokeElement)
	if !ok || poke.Target != "10001" {
		t.Fatalf("unexpected poke segment: %#v", segments[6])
	}

	unknown, ok := segments[7].(*types.TextElement)
	if !ok || !strings.Contains(unknown.Content, "[CQ:mface") {
		t.Fatalf("unexpected unknown fallback segment: %#v", segments[7])
	}
}

func TestOB11BuildMessageSkipsInlinePoke(t *testing.T) {
	pa := &PlatformAdapterOB11{}
	payload := pa.buildMessage([]types.IMessageElement{
		&types.TextElement{Content: "hello"},
		&types.PokeElement{Target: "10001"},
	})

	if len(payload) != 1 {
		t.Fatalf("expected 1 serialized segment, got %d", len(payload))
	}
	if payload[0]["type"] != "text" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestOB11ExtractSegmentsKeepsPlainText(t *testing.T) {
	pa := &PlatformAdapterOB11{}
	segments := pa.extractSegments([]byte(`"prefix[CQ:at,qq=12345]suffix"`))

	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	if text, ok := segments[0].(*types.TextElement); !ok || text.Content != "prefix[CQ:at,qq=12345]suffix" {
		t.Fatalf("unexpected text segment: %#v", segments[0])
	}
}
