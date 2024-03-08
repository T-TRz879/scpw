package scpw

import (
	"fmt"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type Progress struct {
	*mpb.Progress
	bars []*mpb.Bar
}

func NewProgress() *Progress {
	return &Progress{
		mpb.New(mpb.WithWidth(64)),
		[]*mpb.Bar{},
	}
}

func (p *Progress) NewInfiniteByesBar(name string) *mpb.Bar {
	// new bar with 'trigger complete event' disabled, because total is zero
	bar := p.AddBar(0,
		mpb.PrependDecorators(decor.Counters(decor.SizeB1024(0), fmt.Sprintf("%-35s", name)+" | % .1f / % .1f")),
		mpb.AppendDecorators(decor.Percentage()),
	)
	p.bars = append(p.bars, bar)
	return bar
}
