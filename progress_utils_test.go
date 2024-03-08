package scpw

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewProgress(t *testing.T) {
	progress := NewProgress()
	bar := progress.NewInfiniteByesBar("")
	assert.True(t, bar.IsRunning())
}
