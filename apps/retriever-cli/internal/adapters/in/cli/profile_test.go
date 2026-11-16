package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// profileRequiredStub is the setup error the composition root returns when profiles exist and none is chosen. It
// carries the stable code, so the test proves what a caller concludes.
type profileRequiredStub struct{}

func (profileRequiredStub) Error() string {
	return "no profile was selected: pass --profile <name> or set RETRIEVER_PROFILE"
}

func (profileRequiredStub) ErrorCode() string { return "retriever.usage.profile_required" }

// TestParseGlobalsReadsTheProfileFlag checks that --profile is a global option: it reads before, after, and between
// subcommand names, and in both the space and the equals form.
func TestParseGlobalsReadsTheProfileFlag(t *testing.T) {
	cases := []struct {
		argv []string
		want string
	}{
		{[]string{"--profile", "personal", "sources", "list"}, "personal"},
		{[]string{"sources", "--profile", "personal", "list"}, "personal"},
		{[]string{"sources", "list", "--profile", "personal"}, "personal"},
		{[]string{"--profile=work", "sources", "list"}, "work"},
	}
	for _, tc := range cases {
		g, err := ParseGlobals(tc.argv)
		if err != nil || g.Profile != tc.want {
			t.Fatalf("ParseGlobals(%q) = %+v, %v; want profile %q", tc.argv, g, err, tc.want)
		}
	}
}

// TestRepeatedProfileFlagIsAUsageError checks F2: a repeated profile flag exits 2 with nothing on standard output,
// because one run takes exactly one profile.
func TestRepeatedProfileFlagIsAUsageError(t *testing.T) {
	setup := outlineSetup{service: twoSourceService(t), sink: newMemSink()}
	code, out, errOut := runRetriever(t, []string{"--profile", "work", "--profile", "personal", "sources", "list"}, setup)
	if code != 2 || out != "" || !strings.Contains(errOut, "--profile") {
		t.Fatalf("want exit 2 naming --profile with no rows, got stdout %q stderr %q (exit %d)", out, errOut, code)
	}
}

// TestSourcesListNamesTheActiveProfile checks F4: the sources list names the profile that answered, in text and in
// JSON, so a caller sees that no second profile was merged in.
func TestSourcesListNamesTheActiveProfile(t *testing.T) {
	svc := twoSourceService(t)
	svc.Profile = "personal"
	setup := outlineSetup{service: svc, sink: newMemSink()}

	code, out, _ := runRetriever(t, []string{"sources", "list"}, setup)
	if code != 0 || !strings.HasPrefix(out, "2 sources (profile: personal)\n") {
		t.Fatalf("sources list = %q (exit %d), want the personal profile in the count line", out, code)
	}

	code, out, _ = runRetriever(t, []string{"sources", "list", "--output", "json"}, setup)
	var doc struct {
		SchemaVersion int    `json:"schemaVersion"`
		Profile       string `json:"profile"`
		Sources       []struct {
			ID string `json:"id"`
		} `json:"sources"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Profile != "personal" || len(doc.Sources) != 2 {
		t.Fatalf("sources list JSON = %q (exit %d), want the personal profile with two sources", out, code)
	}
}

// TestSourcesListWithoutProfileKeepsTheOldShape checks that a run with no profiles folder prints the list as before:
// no profile in the count line and no profile field in the JSON document.
func TestSourcesListWithoutProfileKeepsTheOldShape(t *testing.T) {
	setup := outlineSetup{service: twoSourceService(t), sink: newMemSink()}

	code, out, _ := runRetriever(t, []string{"sources", "list"}, setup)
	if code != 0 || !strings.HasPrefix(out, "2 sources\n") || strings.Contains(out, "profile") {
		t.Fatalf("sources list = %q (exit %d), want the old count line with no profile", out, code)
	}

	code, out, _ = runRetriever(t, []string{"sources", "list", "--output", "json"}, setup)
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("sources list JSON = %q (exit %d)", out, code)
	}
	if _, ok := doc["profile"]; ok {
		t.Fatalf("sources list JSON names a profile without one: %q", out)
	}
}

// TestProfileRequiredSetupErrorExitsTwoWithItsCode checks F3 at the command surface: a setup error with the
// profile_required code exits 2, keeps standard output empty, and names the code in JSON.
func TestProfileRequiredSetupErrorExitsTwoWithItsCode(t *testing.T) {
	setup := outlineSetup{setupErr: profileRequiredStub{}, sink: newMemSink()}

	code, out, _ := runRetriever(t, []string{"sources", "list"}, setup)
	if code != 2 || out != "" {
		t.Fatalf("want exit 2 with empty stdout, got stdout %q (exit %d)", out, code)
	}
	code, out, errOut := runRetriever(t, []string{"sources", "list", "--output", "json"}, setup)
	var doc struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if code != 2 || out != "" || json.Unmarshal([]byte(errOut), &doc) != nil ||
		doc.Error.Code != "retriever.usage.profile_required" {
		t.Fatalf("want the profile_required body, got stdout %q stderr %q (exit %d)", out, errOut, code)
	}
}
