package errdefs

import (
	"fmt"
	io "io"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type VertexError struct {
	Vertex
	error
}

func Traces(err error) []*Stack {
	var st []*Stack

	wrapped, ok := err.(interface {
		Unwrap() error
	})
	if ok {
		st = Traces(wrapped.Unwrap())
	}

	if ste, ok := err.(interface {
		StackTrace() errors.StackTrace
	}); ok {
		st = append(st, convertStack(ste.StackTrace()))
	}

	if ste, ok := err.(interface {
		StackTrace() *Stack
	}); ok {
		st = append(st, ste.StackTrace())
	}

	return st
}

func StackFormatter(err error) fmt.Formatter {
	return &stackFormatter{err}
}

type stackFormatter struct {
	error
}

func (w *stackFormatter) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", w.Error())
			for _, stack := range Traces(w.error) {
				fmt.Fprintf(s, "%d %s\n", stack.Pid, strings.Join(stack.Cmdline, " "))
				for _, f := range stack.Frames {
					fmt.Fprintf(s, "%s\n\t%s:%d\n", f.Name, f.File, f.Line)
				}
				fmt.Fprintf(s, "\n")
			}
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

func convertStack(s errors.StackTrace) *Stack {
	var out Stack
	for _, f := range s {
		dt, err := f.MarshalText()
		if err != nil {
			continue
		}
		p := strings.SplitN(string(dt), " ", 2)
		if len(p) != 2 {
			continue
		}
		idx := strings.LastIndexByte(p[1], ':')
		if idx == -1 {
			continue
		}
		line, err := strconv.Atoi(p[1][idx+1:])
		if err != nil {
			continue
		}
		out.Frames = append(out.Frames, &Frame{
			Name: p[0],
			File: p[1][:idx],
			Line: int32(line),
		})
	}
	out.Cmdline = os.Args
	out.Pid = int32(os.Getpid())
	return &out
}

type withStack struct {
	stack Stack
	error
}

func (e *withStack) Unwrap() error {
	return e.error
}

func (e *withStack) StackTrace() *Stack {
	return &e.stack
}
