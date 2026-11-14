package outlinestub

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// panicOf runs f and returns the text of the panic it raised, if any.
func panicOf(f func()) (message string, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			message, panicked = fmt.Sprint(r), true
		}
	}()
	f()
	return "", false
}

// unreadableFS opens every file of files, and it fails every read of the named file. It is an in-memory file system,
// so the test reads no disk.
type unreadableFS struct {
	files      fstest.MapFS
	unreadable string
}

var errFixtureUnreadable = errors.New("fixture cannot be read")

func (f unreadableFS) Open(name string) (fs.File, error) {
	file, err := f.files.Open(name)
	if err != nil || name != f.unreadable {
		return file, err
	}
	return brokenFile{file}, nil
}

// brokenFile is a file whose reads fail.
type brokenFile struct {
	fs.File
}

func (brokenFile) Read([]byte) (int, error) { return 0, errFixtureUnreadable }

func TestLoadFixturesDecodesEachFileUnderItsNameWithoutTheExtension(t *testing.T) {
	fsys := fstest.MapFS{
		"testdata/one.json": {Data: []byte(`{"a":1}`)},
		"testdata/two.json": {Data: []byte(`{"b":"x"}`)},
	}

	got := loadFixtures(fsys, "testdata")

	if len(got) != 2 {
		t.Fatalf("loaded %d fixtures, want 2", len(got))
	}
	if got["one"]["a"] != float64(1) || got["two"]["b"] != "x" {
		t.Fatalf("fixtures = %v, want each file decoded under its own name", got)
	}
}

func TestLoadFixturesStopsWhenTheDirectoryCannotBeListed(t *testing.T) {
	msg, panicked := panicOf(func() { loadFixtures(fstest.MapFS{}, "testdata") })

	if !panicked || !strings.HasPrefix(msg, "outlinestub: read fixtures:") {
		t.Fatalf("loadFixtures with no directory panicked %v with %q, want the read failure", panicked, msg)
	}
}

func TestLoadFixturesStopsWhenAFixtureCannotBeRead(t *testing.T) {
	fsys := unreadableFS{
		files: fstest.MapFS{
			"testdata/one.json": {Data: []byte(`{"a":1}`)},
		},
		unreadable: "testdata/one.json",
	}

	msg, panicked := panicOf(func() { loadFixtures(fsys, "testdata") })

	if !panicked || !strings.Contains(msg, "read fixture one.json") || !strings.Contains(msg, errFixtureUnreadable.Error()) {
		t.Fatalf("an unreadable fixture panicked %v with %q, want the read failure of one.json", panicked, msg)
	}
}

func TestLoadFixturesStopsOnAFixtureThatIsNotJSON(t *testing.T) {
	fsys := fstest.MapFS{
		"testdata/broken.json": {Data: []byte(`{"a":`)},
	}

	msg, panicked := panicOf(func() { loadFixtures(fsys, "testdata") })

	if !panicked || !strings.Contains(msg, "decode fixture broken.json") {
		t.Fatalf("a broken fixture panicked %v with %q, want the decode failure of broken.json", panicked, msg)
	}
}

func TestEmbeddedFixturesAreAllLoaded(t *testing.T) {
	for _, name := range []string{"collections.list", "documents.info", "error-401", "error-500"} {
		if _, ok := fixtures[name]; !ok {
			t.Errorf("the embedded fixture %q is not loaded", name)
		}
	}
}

func TestFixtureTopStopsOnAnUnknownFixture(t *testing.T) {
	msg, panicked := panicOf(func() { fixtureTop("no-such-fixture") })

	if !panicked || msg != `outlinestub: unknown fixture "no-such-fixture"` {
		t.Fatalf("fixtureTop panicked %v with %q, want the unknown fixture", panicked, msg)
	}
}

func TestFixtureObjectReadsAnObjectOrOneElementOfAList(t *testing.T) {
	first := fixtureObject("collections.list", "data", 0)
	if first["id"] != "fixture-collection-0001" {
		t.Fatalf("element 0 = %v, want the fixture collection", first)
	}

	page := fixtureObject("collections.list", "pagination", -1)
	if _, ok := page["offset"]; !ok {
		t.Fatalf("object read = %v, want the pagination object", page)
	}
}

func TestFixtureObjectStopsOnAKeyOrElementThatIsNotThere(t *testing.T) {
	fixtures["scalar-list"] = map[string]any{"items": []any{"text"}, "name": "plain"}
	t.Cleanup(func() { delete(fixtures, "scalar-list") })

	cases := []struct {
		name    string
		fixture string
		key     string
		index   int
		want    string
	}{
		{name: "a missing key", fixture: "collections.list", key: "absent", index: 0,
			want: `outlinestub: fixture "collections.list" lacks "absent"`},
		{name: "an object read of a list", fixture: "collections.list", key: "data", index: -1,
			want: `outlinestub: fixture "collections.list" key "data" is not an object`},
		{name: "a list element past the end", fixture: "collections.list", key: "data", index: 99,
			want: `outlinestub: fixture "collections.list" key "data" lacks element 99`},
		{name: "a list read of a scalar", fixture: "scalar-list", key: "name", index: 0,
			want: `outlinestub: fixture "scalar-list" key "name" lacks element 0`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, panicked := panicOf(func() { fixtureObject(tc.fixture, tc.key, tc.index) })
			if !panicked || msg != tc.want {
				t.Fatalf("fixtureObject panicked %v with %q, want %q", panicked, msg, tc.want)
			}
		})
	}
}

func TestFixtureObjectStopsOnAListElementThatIsNotAnObject(t *testing.T) {
	fixtures["scalar-list"] = map[string]any{"items": []any{"text"}}
	t.Cleanup(func() { delete(fixtures, "scalar-list") })

	msg, panicked := panicOf(func() { fixtureObject("scalar-list", "items", 0) })

	if !panicked || msg != `outlinestub: fixture "scalar-list" element is not an object` {
		t.Fatalf("fixtureObject on a scalar element panicked %v with %q", panicked, msg)
	}
}

func TestJSONReplyStopsOnAValueThatCannotBeEncoded(t *testing.T) {
	msg, panicked := panicOf(func() { jsonReply(200, map[string]any{"channel": make(chan int)}, nil) })

	if !panicked || !strings.HasPrefix(msg, "outlinestub: encode reply:") {
		t.Fatalf("jsonReply panicked %v with %q, want the encode failure", panicked, msg)
	}
}

func TestJSONReplySetsTheContentTypeAndKeepsAGivenHeader(t *testing.T) {
	header := map[string][]string{"Retry-After": {"4"}}

	rep := jsonReply(202, map[string]any{"ok": true}, header)

	if rep.status != 202 || string(rep.body) != `{"ok":true}` {
		t.Fatalf("reply = %d %s, want 202 {\"ok\":true}", rep.status, rep.body)
	}
	if rep.header.Get("Content-Type") != "application/json" || rep.header.Get("Retry-After") != "4" {
		t.Fatalf("headers = %v, want the JSON type and the given Retry-After", rep.header)
	}
}

func TestErrorBodyUsesItsOwnFixtureOrTheGenericOneWithTheStatusReplaced(t *testing.T) {
	own := errorBody(401)
	if own["status"] != 401 || own["message"] != "Authentication required" || own["ok"] != false {
		t.Fatalf("error body for 401 = %v, want its own fixture with status 401", own)
	}

	generic := errorBody(418)
	if generic["status"] != 418 {
		t.Fatalf("error body for 418 = %v, want the status replaced", generic)
	}
	if generic["message"] != fixtureTop("error-500")["message"] {
		t.Fatalf("error body for 418 has message %v, want the generic 500 message", generic["message"])
	}
}

func TestErrorBodyDoesNotChangeTheFixtureItCopies(t *testing.T) {
	before := fixtureTop("error-401")["status"]

	errorBody(401)["status"] = 999

	if fixtureTop("error-401")["status"] != before {
		t.Fatalf("changing a built error body changed the fixture to %v", fixtureTop("error-401")["status"])
	}
}

func TestEndpointNotFoundCarriesTheRouterMessageAndStatus(t *testing.T) {
	rep := endpointNotFound()

	if rep.status != 404 {
		t.Fatalf("status = %d, want 404", rep.status)
	}
	if !strings.Contains(string(rep.body), `"message":"Endpoint not found"`) {
		t.Fatalf("body = %s, want the router's message", rep.body)
	}
}

func TestWindowClampsAPageToTheHeldItems(t *testing.T) {
	cases := []struct {
		total, offset, limit int
		wantStart, wantEnd   int
	}{
		{total: 3, offset: 0, limit: 2, wantStart: 0, wantEnd: 2},
		{total: 3, offset: 2, limit: 5, wantStart: 2, wantEnd: 3},
		{total: 3, offset: 5, limit: 1, wantStart: 3, wantEnd: 3},
		{total: 0, offset: 0, limit: 25, wantStart: 0, wantEnd: 0},
	}
	for _, tc := range cases {
		start, end := window(tc.total, tc.offset, tc.limit)
		if start != tc.wantStart || end != tc.wantEnd {
			t.Errorf("window(%d, %d, %d) = %d, %d; want %d, %d",
				tc.total, tc.offset, tc.limit, start, end, tc.wantStart, tc.wantEnd)
		}
	}
}

func TestCloneObjectCopiesTheTopLevelOnly(t *testing.T) {
	template := map[string]any{"a": 1, "b": "two"}

	clone := cloneObject(template)
	clone["a"] = 99
	clone["c"] = true

	if template["a"] != 1 || len(template) != 2 {
		t.Fatalf("changing the clone changed the template: %v", template)
	}
	if clone["b"] != "two" {
		t.Fatalf("the clone lost a key: %v", clone)
	}
}
