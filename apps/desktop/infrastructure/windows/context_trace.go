package windows

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	contextTracePrefix        = "ARION_CONTEXT_PHASE "
	contextTraceGoReceived    = contextTracePrefix + "go-received"
	contextTraceReadEntered   = contextTracePrefix + "read-entered"
	contextTraceRootStarting  = contextTracePrefix + "root-starting"
	contextTraceRootReady     = contextTracePrefix + "root-ready"
	contextTraceWalkFinished  = contextTracePrefix + "walk-finished"
	contextTraceOutputFlushed = contextTracePrefix + "output-flushed"
)

type contextStage uint8

const (
	stageGoWritten contextStage = iota
	stageGoReceived
	stageReadEntered
	stageRootStarting
	stageRootReady
	stageWalkFinished
	stageOutputFlushed
	stagePayloadSeen
	stageStdoutEOF
	stageProcessExited
	contextStageCount
)

var contextStageNames = [contextStageCount]string{
	"go-written",
	contextTraceGoReceived[len(contextTracePrefix):],
	contextTraceReadEntered[len(contextTracePrefix):],
	contextTraceRootStarting[len(contextTracePrefix):],
	contextTraceRootReady[len(contextTracePrefix):],
	contextTraceWalkFinished[len(contextTracePrefix):],
	contextTraceOutputFlushed[len(contextTracePrefix):],
	"payload-seen",
	"stdout-eof",
	"process-exited",
}

type contextTraceRecord struct {
	stage contextStage
	wire  string
}

var contextTraceRecords = func() [2 * (stageOutputFlushed - stageGoReceived + 1)]contextTraceRecord {
	var records [2 * (stageOutputFlushed - stageGoReceived + 1)]contextTraceRecord
	for stage := stageGoReceived; stage <= stageOutputFlushed; stage++ {
		index := int(stage-stageGoReceived) * 2
		wire := contextTracePrefix + contextStageNames[stage]
		records[index] = contextTraceRecord{stage, wire + "\n"}
		records[index+1] = contextTraceRecord{stage, wire + "\r\n"}
	}
	return records
}()

type contextObservation struct {
	started    time.Time
	events     [contextStageCount]time.Time
	exitFailed bool
}

func (observation contextObservation) String() string {
	var result strings.Builder
	for stage, event := range observation.events {
		if event.IsZero() {
			continue
		}
		if result.Len() > 0 {
			result.WriteByte(' ')
		}
		fmt.Fprintf(&result, "%s=%dms", contextStageNames[stage], event.Sub(observation.started).Milliseconds())
	}
	if !observation.events[stageProcessExited].IsZero() {
		fmt.Fprintf(&result, " exit-failed=%t", observation.exitFailed)
	}
	if result.Len() == 0 {
		return "none"
	}
	return result.String()
}

// Only finite marker candidates and timestamps are retained. Raw stderr bytes,
// including unknown or arbitrarily long lines, are never stored.
type contextTrace struct {
	mu          sync.Mutex
	observation contextObservation
	cancelledAt time.Time
	candidates  uint16
	offset      int
}

func newContextTrace() *contextTrace {
	return &contextTrace{
		observation: contextObservation{started: time.Now()},
		candidates:  (1 << len(contextTraceRecords)) - 1,
	}
}

func (trace *contextTrace) record(stage contextStage) {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	trace.recordLocked(stage)
}

func (trace *contextTrace) recordLocked(stage contextStage) {
	if trace.observation.events[stage].IsZero() {
		trace.observation.events[stage] = time.Now()
	}
}

func (trace *contextTrace) exited(err error) {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	trace.recordLocked(stageProcessExited)
	trace.observation.exitFailed = err != nil
}

func (trace *contextTrace) cancelling() {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	if trace.cancelledAt.IsZero() {
		trace.cancelledAt = time.Now()
	}
}

func (trace *contextTrace) Write(data []byte) (int, error) {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	for _, current := range data {
		for index, record := range contextTraceRecords {
			bit := uint16(1 << index)
			if trace.candidates&bit != 0 && (trace.offset >= len(record.wire) || record.wire[trace.offset] != current) {
				trace.candidates &^= bit
			}
		}
		if current == '\n' {
			for index, record := range contextTraceRecords {
				if trace.candidates&uint16(1<<index) != 0 && trace.offset+1 == len(record.wire) {
					trace.recordLocked(record.stage)
				}
			}
			trace.candidates = (1 << len(contextTraceRecords)) - 1
			trace.offset = 0
		} else if trace.candidates != 0 {
			trace.offset++
		}
	}
	return len(data), nil
}

func (trace *contextTrace) snapshot(cutoff time.Time) contextObservation {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	// cancelling() is recorded before CommandContext kills the child.
	// Windows can stamp that kill's stdout EOF in the same instant, so an
	// event that is not strictly earlier is cleanup, not prior progress.
	onCancel := !trace.cancelledAt.IsZero() && trace.cancelledAt.Before(cutoff)
	if onCancel {
		cutoff = trace.cancelledAt
	}
	observation := trace.observation
	for stage, event := range observation.events {
		drop := event.After(cutoff)
		if onCancel && !event.IsZero() && !event.Before(cutoff) {
			drop = true
		}
		if drop {
			observation.events[stage] = time.Time{}
		}
	}
	if observation.events[stageProcessExited].IsZero() {
		observation.exitFailed = false
	}
	return observation
}

type contextReadFailure struct {
	cause       error
	observation contextObservation
}

func (failure *contextReadFailure) Error() string {
	return fmt.Sprintf("%v (context trace: %s)", failure.cause, failure.observation)
}

func (failure *contextReadFailure) Unwrap() error { return failure.cause }

type observedContextReader struct {
	reader io.Reader
	trace  *contextTrace
}

func (reader observedContextReader) Read(data []byte) (int, error) {
	count, err := reader.reader.Read(data)
	if count > 0 {
		reader.trace.record(stagePayloadSeen)
	}
	if err == io.EOF {
		reader.trace.record(stageStdoutEOF)
	}
	return count, err
}
