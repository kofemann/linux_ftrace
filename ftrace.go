package ftrace

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	FTRACE_SWITCH_FILE    = "/sys/kernel/debug/tracing/tracing_on"
	EVENT_SWITCH_FILE_FMT = "/sys/kernel/debug/tracing/events/%s/enable"
	TRACE_FILE            = "/sys/kernel/debug/tracing/trace_pipe"
)

// Trace represents tracing event.
type Trace struct {
	Task         string
	Pid          int
	Cpu          int
	IrqsOff      bool
	NeedResched  bool
	IsIrq        bool
	PreemptDepth bool
	TimeStamp    int64
	Function     string
	Event        string
}

func (trace *Trace) String() string {
	return fmt.Sprintf("%s %d %d %s: %s",
		trace.Task,
		trace.Pid,
		trace.Cpu,
		trace.Function,
		trace.Event)
}

// EventTrace represents event based tracing.
type EventTrace struct {
	traceFile string
	traces    chan Trace
	pipe      *os.File
}

// NewEventTrace returns a new event tracee for a given type.
func NewEventTrace(trace string) *EventTrace {

	traceFile := fmt.Sprintf(EVENT_SWITCH_FILE_FMT, trace)
	traces := make(chan Trace, 1000)
	return &EventTrace{traceFile: traceFile, traces: traces}

}

// Enable enables tracing. Returns error, if failed.
func (ftrace *EventTrace) Enable() error {

	// enable tracing facility
	err := ioutil.WriteFile(FTRACE_SWITCH_FILE, []byte("1"), 0600)
	if err != nil {
		return err
	}

	// enable tracing for a given event
	err = ioutil.WriteFile(ftrace.traceFile, []byte("1"), 0600)
	if err != nil {
		ioutil.WriteFile(FTRACE_SWITCH_FILE, []byte("0"), 0600)
		return err
	}

	ftrace.pipe, err = os.Open(TRACE_FILE)
	if err != nil {
		ioutil.WriteFile(FTRACE_SWITCH_FILE, []byte("0"), 0600)
		return err
	}
	return nil
}

// Disable disables tracing.
func (ftrace *EventTrace) Disable() error {

	close(ftrace.traces)
	ftrace.pipe.Close()
	ioutil.WriteFile(ftrace.traceFile, []byte("0"), 0600)
	ioutil.WriteFile(FTRACE_SWITCH_FILE, []byte("0"), 0600)
	return nil
}

// EventSource returns a channel of trace events.
func (ftrace *EventTrace) EventSource() chan Trace {

	go func() {
		bufferReader := bufio.NewReader(ftrace.pipe)
		for {
			s, err := bufferReader.ReadString('\n')
			if err != nil {
				break
			}
			if len(s) == 0 || s[0] == '#' {
				continue
			}
			t, err := toTrace(s)
			if err == nil {
				ftrace.traces <- t
			}
		}
	}()

	return ftrace.traces
}

// sinitize trace string: remove leading, trailing and multiple white spaces
func sanitize(s string) string {
	return strings.Replace(strings.TrimSpace(s), "  ", " ", -1)
}

// convert trace string into trace event
func toTrace(s string) (Trace, error) {

	var t Trace
	fields := strings.SplitN(sanitize(s), " ", 6)
	if len(fields) != 6 {
		return t, fmt.Errorf("Unexpected number of fields in trace: %d", len(fields))
	}

	taskPid := strings.Split(fields[0], "-")
	if len(taskPid) != 2 {
		return t, fmt.Errorf("Unexpected number of fields in task: %d", len(taskPid))
	}

	pid, err := strconv.Atoi(taskPid[1])
	if err != nil {
		return t, err
	}

	cpu, err := strconv.Atoi(fields[1][1:4])
	if err != nil {
		return t, err
	}

	t = Trace{
		Task:     taskPid[0],
		Pid:      pid,
		Cpu:      cpu,
		Function: fields[4][:len(fields[4])-1],
		Event:    fields[5],
	}
	return t, nil
}
