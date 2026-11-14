package outline

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// seamEnv returns a getenv that reads only the given variables.
func seamEnv(vars map[string]string) func(string) string {
	return func(name string) string {
		return vars[name]
	}
}

func TestSeamIsOffWhenNeitherVariableIsSet(t *testing.T) {
	seam, err := seamFromEnvNoRead(seamEnv(map[string]string{}))
	if err != nil {
		t.Fatalf("TestSeamFromEnv error = %v, want nil when unset", err)
	}
	if seam.Enabled || seam.BaseURL != "" || seam.Client != nil {
		t.Fatalf("seam = %+v, want off", seam)
	}
}

// Every case below is refused before the CA file is read, so no file is touched.
func TestSeamRefusesAPlainHTTPOverride(t *testing.T) {
	seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
		BaseURLEnv: "http://127.0.0.1:8080",
		CAFileEnv:  "ca.pem",
	}))
	if !errors.Is(err, domain.ErrHTTPSRequired) {
		t.Fatalf("error = %v, want ErrHTTPSRequired", err)
	}
	if seam.Enabled {
		t.Fatalf("seam accepted a plain HTTP override")
	}
}

func TestSeamRefusesAnyHostThatIsNotLoopback(t *testing.T) {
	for _, base := range []string{
		"https://docs.example.test",
		"https://example.com",
		"https://198.51.100.5:8443",
		"https://0.0.0.0:8443",
		"https://[2001:db8::1]:8443",
		"https://localhost.example.com",
		"https://",
	} {
		t.Run(base, func(t *testing.T) {
			seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
				BaseURLEnv: base,
				CAFileEnv:  "ca.pem",
			}))
			if err == nil {
				t.Fatalf("seam accepted %q, want an error", base)
			}
			if seam.Enabled {
				t.Fatalf("seam reported enabled for %q", base)
			}
		})
	}
}

// TestSeamRefusesTheLocalhostName checks that only an IP literal counts as loopback. The name localhost is refused
// with the loopback error, so the seam never depends on how a name resolves on the machine.
func TestSeamRefusesTheLocalhostName(t *testing.T) {
	for _, base := range []string{"https://localhost:8443", "https://LOCALHOST:8443"} {
		t.Run(base, func(t *testing.T) {
			seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
				BaseURLEnv: base,
				CAFileEnv:  "ca.pem",
			}))
			if !errors.Is(err, errSeamNotLoopback) || seam.Enabled {
				t.Fatalf("base %q: enabled=%v err=%v, want errSeamNotLoopback", base, seam.Enabled, err)
			}
		})
	}
}

func TestSeamRefusesABaseWithoutACAFile(t *testing.T) {
	seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
		BaseURLEnv: "https://127.0.0.1:8443",
	}))
	if err == nil || seam.Enabled {
		t.Fatalf("seam accepted a base with no CA file: enabled=%v err=%v", seam.Enabled, err)
	}
}

func TestSeamRefusesACAFileWithoutABase(t *testing.T) {
	seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
		CAFileEnv: "ca.pem",
	}))
	if err == nil || seam.Enabled {
		t.Fatalf("seam accepted a CA file with no base: enabled=%v err=%v", seam.Enabled, err)
	}
}

func TestSeamRefusesUserInfoInTheBase(t *testing.T) {
	seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
		BaseURLEnv: "https://reader@127.0.0.1:8443",
		CAFileEnv:  "ca.pem",
	}))
	if err == nil || seam.Enabled {
		t.Fatalf("seam accepted user information in the base: enabled=%v err=%v", seam.Enabled, err)
	}
}

// A timeout alone bounds nothing, so it is refused before any other check.
func TestSeamRefusesATimeoutWithoutABase(t *testing.T) {
	seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
		TimeoutEnv: "500ms",
	}))
	if !errors.Is(err, errSeamTimeoutNeedsBase) || seam.Enabled {
		t.Fatalf("seam with a timeout and no base: enabled=%v err=%v, want errSeamTimeoutNeedsBase", seam.Enabled, err)
	}
}

// Every bad timeout is refused before the CA file is read, so no file is touched.
func TestSeamRefusesATimeoutThatIsNotAPositiveDuration(t *testing.T) {
	for _, value := range []string{"soon", "500", "0s", "-1s", "1 second"} {
		t.Run(value, func(t *testing.T) {
			seam, err := seamFromEnvNoRead(seamEnv(map[string]string{
				BaseURLEnv: "https://127.0.0.1:8443",
				CAFileEnv:  "ca.pem",
				TimeoutEnv: value,
			}))
			if !errors.Is(err, errSeamTimeoutInvalid) || seam.Enabled {
				t.Fatalf("timeout %q: enabled=%v err=%v, want errSeamTimeoutInvalid", value, seam.Enabled, err)
			}
		})
	}
}

// TestSeamTimeoutRule checks the duration rule on its own, with no file or listener.
func TestSeamTimeoutRule(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "unset keeps the default timeout", value: "", want: 0},
		{name: "milliseconds", value: "500ms", want: 500 * time.Millisecond},
		{name: "whole seconds", value: "2s", want: 2 * time.Second},
		{name: "zero is not positive", value: "0s", wantErr: true},
		{name: "negative is not positive", value: "-1s", wantErr: true},
		{name: "a bare number has no unit", value: "500", wantErr: true},
		{name: "words are not a duration", value: "soon", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSeamTimeout(tc.value)
			if tc.wantErr {
				if !errors.Is(err, errSeamTimeoutInvalid) {
					t.Fatalf("parseSeamTimeout(%q) error = %v, want errSeamTimeoutInvalid", tc.value, err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("parseSeamTimeout(%q) = %v, %v, want %v, nil", tc.value, got, err, tc.want)
			}
		})
	}
}

// seamFromEnvNoRead reads the seam through getenv with a CA reader that always fails, so a unit test reads no file. A
// setting that reaches the CA read gets the failure and never touches the disk.
func seamFromEnvNoRead(getenv func(string) string) (Seam, error) {
	return TestSeamFromEnvWithFiles(getenv, func(string) ([]byte, error) { return nil, os.ErrNotExist })
}
