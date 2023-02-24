package booter_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

func TestFlagParser(t *testing.T) {
	f := &flags{args: []string{
		"--strlongsep", `"strval"`,
		"--strlong='str val'",
		"--exists",
		"-flag=true",
	}}
	for {
		ok, err := f.parseOne()
		if err != nil {
			panic(err)
		}
		if !ok {
			break
		}

		if f.hasValue {
			t.Logf("name:'%s' value:'%s' single-minus:%t", f.name, f.value, f.hasSingleDash)
		} else {
			t.Logf("name:'%s' exists single-minux:%t", f.name, f.hasSingleDash)
		}
	}
}

type flags struct {
	args          []string
	name          string
	value         string
	hasValue      bool
	hasSingleDash bool
}

func (f *flags) failf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func (f *flags) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.failf("bad flag syntax: %s", s)
	}

	if numMinuses == 1 {
		f.hasSingleDash = true
	}

	// it's a flag. does it have an argument?
	f.args = f.args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}

	// It must have a value, which might be the next argument.
	if !hasValue && len(f.args) > 0 && !strings.HasPrefix(f.args[0], "-") {
		// value is the next arg
		hasValue = true
		value, f.args = f.args[0], f.args[1:]
	}

	f.hasValue = hasValue
	if hasValue {
		f.name = name
		f.value = StripQuote(value)
	}
	return true, nil
}

func StripQuote(str string) string {
	if len(str) == 0 {
		return str
	}
	c := []rune(str)[0]
	if unicode.In(c, unicode.Quotation_Mark) {
		return strings.Trim(str, string(c))
	}
	return str
}
