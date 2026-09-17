package taskmonitor

import (
	"context"
	"encoding/json"
	"testing"
)

type testRepository struct{ value Task }

func (r testRepository) List(context.Context, string, Filter) (Page, error) {
	return Page{Items: []Task{r.value}, Total: 1, Page: 1, PageSize: 20, Counts: map[string]int{"queued": 1}}, nil
}
func (r testRepository) Get(context.Context, string, string) (Task, error) { return r.value, nil }

func TestServiceRedactsSensitivePayloadFields(t *testing.T) {
	service := NewService(testRepository{value: Task{Payload: json.RawMessage(`{"runId":"run_1","apiToken":"secret","nested":{"password":"hidden","authorization":"Bearer example"}}`)}})
	value, err := service.Get(context.Background(), "ws_test", "task_test")
	if err != nil {
		t.Fatal(err)
	}
	payload := value.SafePayload.(map[string]any)
	nested := payload["nested"].(map[string]any)
	if payload["runId"] != "run_1" || payload["apiToken"] != "[redacted]" || nested["password"] != "[redacted]" || nested["authorization"] != "[redacted]" {
		t.Fatalf("unexpected safe payload: %#v", payload)
	}
}
