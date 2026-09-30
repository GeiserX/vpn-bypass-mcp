package tools

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// args reads MCP tool arguments. The socket wants every value as a string, so
// numbers and booleans are converted here: 8080 becomes "8080", true becomes
// "true". An empty string counts as absent.
type args map[string]any

func (a args) str(key string) (string, bool, error) {
	raw, ok := a[key]
	if !ok || raw == nil {
		return "", false, nil
	}
	var s string
	switch v := raw.(type) {
	case string:
		s = v
	case bool:
		s = strconv.FormatBool(v)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) {
			return "", true, fmt.Errorf("%s must be a whole number", key)
		}
		if math.Abs(v) >= 1<<53 {
			return "", true, fmt.Errorf("%s is too large", key)
		}
		s = strconv.FormatInt(int64(v), 10)
	case int:
		s = strconv.Itoa(v)
	case int64:
		s = strconv.FormatInt(v, 10)
	default:
		return "", true, fmt.Errorf("%s must be a string, a number or a boolean", key)
	}
	if s == "" {
		return "", false, nil
	}
	return s, true, nil
}

func (a args) required(key string) (string, error) {
	s, ok, err := a.str(key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	return s, nil
}

func (a args) requiredBool(key string) (bool, error) {
	s, err := a.required(key)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(s) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%s must be true or false", key)
}

// wire copies the named MCP arguments into socket args. Each pair is the MCP
// argument name, then the socket argument name.
func (a args) wire(pairs ...string) (map[string]string, error) {
	out := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v, ok, err := a.str(pairs[i])
		if err != nil {
			return nil, err
		}
		if ok {
			out[pairs[i+1]] = v
		}
	}
	return out, nil
}

// secrets returns secrets.pass when a password argument was given.
func (a args) secrets() (map[string]string, error) {
	pass, ok, err := a.str("password")
	if err != nil || !ok {
		return nil, err
	}
	return map[string]string{"pass": pass}, nil
}

// oneOf checks that exactly one of two arguments was given.
func (a args) oneOf(x, y string) error {
	_, hasX, err := a.str(x)
	if err != nil {
		return err
	}
	_, hasY, err := a.str(y)
	if err != nil {
		return err
	}
	switch {
	case hasX && hasY:
		return fmt.Errorf("give %s or %s, not both", x, y)
	case !hasX && !hasY:
		return fmt.Errorf("give %s or %s", x, y)
	}
	return nil
}
