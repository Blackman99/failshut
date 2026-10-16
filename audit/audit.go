// Package audit orchestrates discovery + rules into a report.Result.
package audit

import (
	"fmt"

	"github.com/Blackman99/failshut/discover"
	"github.com/Blackman99/failshut/report"
	"github.com/Blackman99/failshut/rules"
)

// Options for a run.
type Options struct {
	Home    string
	Project string
	Rules   []string // optional filter by rule id; empty = all
}

// Run discovers configs and evaluates every rule.
func Run(opt Options) (report.Result, error) {
	cfgs, err := discover.Discover(discover.Options{Home: opt.Home, Project: opt.Project})
	if err != nil {
		return report.Result{}, err
	}
	res := report.Result{
		Findings:   []report.Finding{},
		Configs:    len(cfgs),
		Configured: len(cfgs) > 0,
	}
	if len(cfgs) == 0 {
		res.Message = "no permission configs found"
		return res, nil
	}

	ctx := rules.Context{Configs: cfgs, Project: opt.Project, Home: opt.Home}
	want := map[string]bool{}
	for _, id := range opt.Rules {
		want[id] = true
	}
	sealedRules := 0
	for _, r := range rules.All() {
		if len(want) > 0 && !want[r.ID()] {
			continue
		}
		fs, err := r.Run(ctx)
		if err != nil {
			return res, fmt.Errorf("%s: %w", r.ID(), err)
		}
		openHere := 0
		for _, f := range fs {
			if f.Severity == "" {
				f.Severity = report.FailOpen
			}
			res.Findings = append(res.Findings, f)
			if f.Severity == report.FailOpen {
				res.FailOpen++
				openHere++
			}
		}
		if openHere == 0 {
			sealedRules++
		}
	}
	res.FailShut = sealedRules
	return res, nil
}
