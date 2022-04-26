package internal

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (p Prompt) question() string {
	v := p.Label
	if p.Label == "" {
		v = p.Var
	}
	return strings.TrimRight(v, "?")
}

func (p Prompt) ask(out io.Writer, in *bufio.Reader) (interface{}, error) {
	rq := request(out, in).Ask(p.question()).Default(p.Default).Options(p.Options)
	switch p.Type {
	case VarInt:
		return rq.Int()
	case VarFloat:
		return rq.Float()
	case VarList:
		return rq.List()
	case VarBool:
		return rq.Bool()
	case "":
		fallthrough
	case VarString:
		return rq.String()
	default:
		return nil, fmt.Errorf("unsupported type %s", p.Type)
	}
}

type requestOpt struct {
	err          error
	options      []string
	out          io.Writer
	in           *bufio.Reader
	defaultValue string
}

func request(out io.Writer, in *bufio.Reader) *requestOpt {
	return &requestOpt{
		out: out,
		in:  in,
	}
}

func (rq *requestOpt) Ask(question string) *requestOpt {
	if rq.err != nil {
		return rq
	}
	if _, err := fmt.Fprintf(rq.out, strings.TrimRight(question, "?")+"? "); err != nil {
		return rq.withError(err)
	}
	return rq
}

func (rq *requestOpt) Options(options []string) *requestOpt {
	if rq.err != nil || len(options) == 0 {
		return rq
	}
	if _, err := fmt.Fprintln(rq.out); err != nil {
		return rq.withError(err)
	}
	for i, opt := range options {
		if _, err := fmt.Fprintln(rq.out, i+1, " - ", opt); err != nil {
			return rq.withError(err)
		}
	}
	rq.options = options
	return rq
}

func (rq *requestOpt) Default(value string) *requestOpt {
	if rq.err != nil || value == "" {
		return rq
	}
	if _, err := fmt.Fprint(rq.out, "[default: ", value, "] "); err != nil {
		return rq.withError(err)
	}
	rq.defaultValue = value
	return rq
}

func (rq *requestOpt) List() ([]string, error) {
	if rq.err != nil {
		return nil, rq.err
	}
	if len(rq.options) > 0 {
		if _, err := fmt.Fprint(rq.out, "Pick options (comma separated): "); err != nil {
			rq.err = err
			return nil, err
		}
		return rq.getOpts()
	}

	if _, err := fmt.Fprint(rq.out, "Enter values (comma separated): "); err != nil {
		rq.err = err
		return nil, err
	}
	return rq.getValues()
}

func (rq *requestOpt) Bool() (bool, error) {
	if rq.err != nil {
		return false, rq.err
	}
	if _, err := fmt.Fprint(rq.out, "(y/n): "); err != nil {
		rq.err = err
		return false, err
	}
	line, err := rq.getSingle()
	if err != nil {
		rq.err = err
		return false, err
	}
	return toBool(line), nil
}

func (rq *requestOpt) String() (string, error) {
	if rq.err != nil {
		return "", rq.err
	}
	if _, err := fmt.Fprint(rq.out, "(text): "); err != nil {
		rq.err = err
		return "", err
	}
	return rq.getSingle()
}

func (rq *requestOpt) Int() (int64, error) {
	if rq.err != nil {
		return 0, rq.err
	}
	if _, err := fmt.Fprint(rq.out, "(integer): "); err != nil {
		rq.err = err
		return 0, err
	}
	line, err := rq.getSingle()
	if err != nil {
		rq.err = err
		return 0, err
	}
	return strconv.ParseInt(line, 10, 64)
}

func (rq *requestOpt) Float() (float64, error) {
	if rq.err != nil {
		return 0, rq.err
	}
	if _, err := fmt.Fprint(rq.out, "(float): "); err != nil {
		rq.err = err
		return 0, err
	}
	line, err := rq.getSingle()
	if err != nil {
		rq.err = err
		return 0, err
	}
	return strconv.ParseFloat(line, 64)
}

func (rq *requestOpt) getOpts() ([]string, error) {
	line, err := rq.getLine()
	if err != nil {
		rq.err = err
		return nil, err
	}
	var values []string
	for _, idx := range strings.Split(line, ",") {
		idx = strings.TrimSpace(idx)
		if idx == "" {
			continue
		}
		v, err := strconv.Atoi(idx)
		if err != nil {
			rq.err = err
			return nil, err
		}
		if v < 1 || v > len(rq.options) {
			return nil, fmt.Errorf("unsupported option %d", v)
		}
		values = append(values, rq.options[v-1])
	}
	return values, nil
}

func (rq *requestOpt) getValues() ([]string, error) {
	data, _, err := rq.in.ReadLine()
	if err != nil {
		rq.err = err
		return nil, err
	}
	return toList(string(data)), nil
}

func (rq *requestOpt) getLine() (string, error) {
	data, _, err := rq.in.ReadLine()
	if err != nil {
		rq.err = err
		return "", err
	}
	line := strings.TrimSpace(string(data))
	if line == "" {
		line = rq.defaultValue
	}
	return line, nil
}

func (rq *requestOpt) getSingle() (string, error) {
	if len(rq.options) > 0 {
		vals, err := rq.getOpts()
		if err != nil {
			return "", err
		}
		if len(vals) == 0 {
			return "", fmt.Errorf("option not picked")
		}
		return vals[0], nil
	}

	return rq.getLine()
}

func (rq *requestOpt) withError(err error) *requestOpt {
	rq.err = err
	return rq
}

func toBool(line string) bool {
	line = strings.ToLower(line)
	return line == "t" || line == "y" || line == "true" || line == "yes" || line == "ok"
}

func toList(line string) []string {
	var values []string
	line = strings.TrimSpace(line)
	for _, value := range strings.Split(line, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}
