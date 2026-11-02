package application

import (
	"context"
	"errors"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

func (c *Coordinator) finishRecognizer(r *run) (contracts.Draft, error) {
	started := time.Now()
	draft, err := r.recognizer.Finish(r.ctx)
	c.diagnostic(r, ports.DiagnosticFinish, started, err)
	return draft, err
}

func (c *Coordinator) diagnostic(r *run, stage ports.DiagnosticStage, start time.Time, err error) {
	if c.deps.Diagnostics == nil {
		return
	}
	code := domain.CodeOf(err)
	if errors.Is(err, context.Canceled) {
		code = domain.CodeCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		code = domain.CodeRewriteTimeout
	}
	if err != nil && code == "" {
		code = domain.CodeContractUnsupported
	}
	c.deps.Diagnostics.Observe(ports.Diagnostic{Identity: r.identity, Stage: stage, Duration: time.Since(start), Failure: code})
}
