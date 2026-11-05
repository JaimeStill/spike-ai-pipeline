package probe_test

import "testing"

// argValue returns the value an argument list gives the flag named by any of names, such as
// --cache-ram and its alias -cram, in either form llama-server reads: --flag value or
// --flag=value. A flag given more than once takes its last value, as llama-server's parser does.
// ok is false when the list doesn't give the flag a value.
func argValue(args []string, names ...string) (v string, ok bool) {
	for i, a := range args {
		for _, n := range names {
			switch {
			case a == n && i+1 < len(args):
				v, ok = args[i+1], true
			case len(a) > len(n) && a[:len(n)+1] == n+"=":
				v, ok = a[len(n)+1:], true
			}
		}
	}
	return v, ok
}

// TestArgValue checks that argValue reads a flag's value in both forms and under its alias, and
// reads nothing from a list without the flag, a flag with no value after it, or a longer flag
// that only starts with the name.
func TestArgValue(t *testing.T) {
	names := []string{"--cache-ram", "-cram"}
	cases := []struct {
		name   string
		args   []string
		want   string
		wantOK bool
	}{
		{"separate", []string{"llama-server", "--cache-ram", "0", "--port", "8081"}, "0", true},
		{"equals", []string{"llama-server", "--cache-ram=0"}, "0", true},
		{"alias", []string{"llama-server", "-cram", "0"}, "0", true},
		{"last wins", []string{"--cache-ram", "8192", "--cache-ram", "0"}, "0", true},
		{"absent", []string{"llama-server", "--ctx-size", "131072", "--parallel", "4"}, "", false},
		{"no value", []string{"llama-server", "--cache-ram"}, "", false},
		{"longer flag", []string{"--cache-ram-max", "0", "--cache-ramx=0"}, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := argValue(c.args, names...)
			if got != c.want || ok != c.wantOK {
				t.Errorf("argValue(%q) = %q, %v, want %q, %v", c.args, got, ok, c.want, c.wantOK)
			}
		})
	}
}
