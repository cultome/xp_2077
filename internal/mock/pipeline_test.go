package mock

import "testing"

func TestPipelineTickCompletes(t *testing.T) {
	p := NewPipeline()
	state := p.State()
	if state.Progress != 0 {
		t.Fatalf("expected initial progress 0, got %d", state.Progress)
	}
	for i := 0; i < 30 && !state.Done; i++ {
		state = p.Tick()
	}
	if !state.Done {
		t.Fatal("expected pipeline to complete")
	}
	if state.Progress != 100 {
		t.Fatalf("expected 100 progress, got %d", state.Progress)
	}
}
