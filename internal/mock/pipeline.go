package mock

import "fmt"

type Pipeline struct {
	stages  []string
	progress int
}

type PipelineState struct {
	StageName string
	StageIdx  int
	StageMax  int
	Progress  int
	Done      bool
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		stages: []string{"fetch", "normalize", "calculateXP"},
	}
}

func (p *Pipeline) Reset() {
	p.progress = 0
}

func (p *Pipeline) Tick() PipelineState {
	if p.progress < 100 {
		p.progress += 5
		if p.progress > 100 {
			p.progress = 100
		}
	}
	return p.State()
}

func (p *Pipeline) State() PipelineState {
	perStage := 100 / len(p.stages)
	idx := p.progress / perStage
	if idx >= len(p.stages) {
		idx = len(p.stages) - 1
	}
	state := PipelineState{
		StageName: p.stages[idx],
		StageIdx:  idx + 1,
		StageMax:  len(p.stages),
		Progress:  p.progress,
		Done:      p.progress >= 100,
	}
	if state.Done {
		state.StageName = "complete"
	}
	return state
}

func (s PipelineState) Label() string {
	return fmt.Sprintf("[%d/%d] %s", s.StageIdx, s.StageMax, s.StageName)
}
