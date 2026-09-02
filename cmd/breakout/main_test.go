package main

import (
	"reflect"
	"testing"
)

func TestBubbleTeaEnvironmentPreventsTmuxHelperExecution(t *testing.T) {
	input := []string{"TERM=screen-256color", "TMUX=/tmp/tmux-1000/default,1,0", "COLORTERM=truecolor", "EMPTY="}
	want := []string{"TERM=screen-256color", "COLORTERM=truecolor", "EMPTY="}
	if got := bubbleTeaEnvironment(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}
