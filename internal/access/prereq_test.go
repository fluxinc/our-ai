package access

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestCommandFailureDecisionExplainsMissingGH(t *testing.T) {
	d := commandFailureDecision(nil, &exec.Error{Name: "gh", Err: exec.ErrNotFound}, "actor_lookup_failed")
	if d.ReasonCode != "gh_missing" || !strings.Contains(d.Message, "gh auth login") || !strings.Contains(d.Message, "not installed") {
		t.Fatalf("decision = %+v", d)
	}
	d = commandFailureDecision([]byte("To get started with GitHub CLI, please run:  gh auth login"), errors.New("exit status 4"), "actor_lookup_failed")
	if d.ReasonCode != "authentication_failed" || !strings.Contains(d.Message, "gh auth login") {
		t.Fatalf("decision = %+v", d)
	}
}
