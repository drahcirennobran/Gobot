package main_test

import (
	"github.com/drahcirennobran/bobot"
	"github.com/drahcirennobran/queue"
	"testing"
)

func TestSplitAcceleration(t *testing.T) {
	command := queue.Command{main.FW, 1000, 2}
	accelerationCommands := main.SplitAcceleration(command)
	if len(accelerationCommands) == 0 {
		t.Errorf("pouet pouet %d", 0)
	}
}
