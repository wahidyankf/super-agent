package outline

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// notAnObject is a reply part that is text where the projection needs an object or a list.
const notAnObject = "not an object"

// TestAReplyPartOfTheWrongShapeIsUnexpected checks each projection and helper that refuses a reply part of the wrong
// shape: a text root, a text item, a list that is not a list, and an object with no ID. Each is an unexpected reply.
func TestAReplyPartOfTheWrongShapeIsUnexpected(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{"dataObject of a text root", func() error { _, err := dataObject(notAnObject); return err }},
		{"dataList of a text root", func() error { _, err := dataList(notAnObject); return err }},
		{"flattenTree of a text node", func() error {
			var out []domain.Record
			return flattenTree([]any{notAnObject}, nil, 0, &out)
		}},
		{"projectExport of a text root", func() error { _, err := projectExport(notAnObject); return err }},
		{"projectAccount of a text root", func() error { _, err := projectAccount(notAnObject); return err }},
		{"projectSearch with a text hit", func() error {
			_, err := projectSearch(map[string]any{"data": []any{notAnObject}})
			return err
		}},
		{"projectSearch with a hit that holds no document", func() error {
			_, err := projectSearch(map[string]any{"data": []any{map[string]any{}}})
			return err
		}},
		{"projectSearch whose data is not a list", func() error {
			_, err := projectSearch(map[string]any{"data": notAnObject})
			return err
		}},
		{"projectDocumentList with a text item", func() error {
			_, err := projectDocumentList(map[string]any{"data": []any{notAnObject}})
			return err
		}},
		{"projectDocumentList with a document that has no ID", func() error {
			_, err := projectDocumentList(map[string]any{"data": []any{map[string]any{}}})
			return err
		}},
		{"projectItems with a text item", func() error {
			_, err := projectItems([]any{notAnObject}, userSpec)
			return err
		}},
		{"projectItems with an item that has no ID", func() error {
			_, err := projectItems([]any{map[string]any{}}, userSpec)
			return err
		}},
		{"projectObjectOf of a text root", func() error { _, err := projectObjectOf(userSpec)(notAnObject); return err }},
		{"projectNestedOf of a text root", func() error { _, err := projectNestedOf("users", userSpec)(notAnObject); return err }},
		{"projectNestedOf whose list is text", func() error {
			_, err := projectNestedOf("users", userSpec)(map[string]any{"data": map[string]any{"users": notAnObject}})
			return err
		}},
		{"projectShareInfo of a text root", func() error { _, err := projectShareInfo(notAnObject); return err }},
		{"projectShareInfo whose first share is text", func() error {
			_, err := projectShareInfo(map[string]any{"data": map[string]any{"shares": []any{notAnObject}}})
			return err
		}},
		{"projectChildren of a text root", func() error { _, err := projectChildren(notAnObject); return err }},
		{"projectChildren whose data is text", func() error {
			_, err := projectChildren(map[string]any{"data": notAnObject})
			return err
		}},
		{"projectChildren whose document has no ID", func() error {
			_, err := projectChildren(map[string]any{"data": map[string]any{}})
			return err
		}},
		{"projectChildren with a text child", func() error {
			_, err := projectChildren(map[string]any{"data": map[string]any{"id": "doc-1", "children": []any{notAnObject}}})
			return err
		}},
		{"projectUnder of a text root", func() error { _, err := projectUnder("object", userSpec)(notAnObject); return err }},
		{"projectSuggestions of a text root", func() error { _, err := projectSuggestions(notAnObject); return err }},
		{"projectSuggestions whose kind is text", func() error {
			_, err := projectSuggestions(map[string]any{"data": map[string]any{"documents": notAnObject}})
			return err
		}},
		{"projectSuggestions with a text item", func() error {
			_, err := projectSuggestions(map[string]any{"data": map[string]any{"documents": []any{notAnObject}}})
			return err
		}},
		{"projectSuggestions with an item that has no ID", func() error {
			_, err := projectSuggestions(map[string]any{"data": map[string]any{"documents": []any{map[string]any{}}}})
			return err
		}},
		{"currentIntegrationSpec of a text root", func() error { _, err := currentIntegrationSpec(notAnObject); return err }},
		{"currentAuthProviderSpec of a text root", func() error { _, err := currentAuthProviderSpec(notAnObject); return err }},
		{"a template with an empty ID", func() error { _, err := templateSpec(map[string]any{"id": ""}); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("error = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

// TestAnEditorBlockSkipsAChildThatIsNotAnObject checks endsBlock: a child that is not an object is passed over, and the
// block still ends its line when its other children are inline.
func TestAnEditorBlockSkipsAChildThatIsNotAnObject(t *testing.T) {
	children := []any{notAnObject, map[string]any{"type": "text"}}
	if !endsBlock(map[string]any{"type": "paragraph"}, children) {
		t.Fatalf("endsBlock = false, want true for a paragraph of inline nodes")
	}
}

// neverDialed is a dialer that a test sets but never calls. A transport with a dialer keeps its TLS settings unset when it
// is copied, which is the case withTLSName's fresh configuration covers.
func neverDialed(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("the test never dials")
}

// TestTheTLSNameIsSetOnATransportWithNoTLSSettings checks withTLSName: a transport with no TLS settings gets a fresh TLS
// configuration that names the host, and the caller's transport does not take the host as its TLS name.
func TestTheTLSNameIsSetOnATransportWithNoTLSSettings(t *testing.T) {
	caller := &http.Transport{DialContext: neverDialed}
	got, ok := withTLSName(caller, "docs.example.test").(*http.Transport)
	if !ok || got.TLSClientConfig == nil || got.TLSClientConfig.ServerName != "docs.example.test" {
		t.Fatalf("withTLSName = %+v; want a transport whose TLS name is the host", got)
	}
	if caller.TLSClientConfig != nil && caller.TLSClientConfig.ServerName == "docs.example.test" {
		t.Fatalf("withTLSName changed the caller's transport")
	}
}

// TestCheckBaseURLRefusesAnUnparsableBase checks that a base the URL parser rejects is an invalid base.
func TestCheckBaseURLRefusesAnUnparsableBase(t *testing.T) {
	if err := CheckBaseURL("https://[::1"); !errors.Is(err, errInvalidBaseURL) {
		t.Fatalf("CheckBaseURL error = %v, want errInvalidBaseURL", err)
	}
}

// TestWithTimeoutSetsTheRequestTimeout checks that the timeout option sets the time limit of one request.
func TestWithTimeoutSetsTheRequestTimeout(t *testing.T) {
	rig := newRig(t, nil, WithTimeout(7*time.Second))
	if rig.gw.timeout != 7*time.Second {
		t.Fatalf("request timeout = %v, want 7s", rig.gw.timeout)
	}
}

// TestEndpointLookupsNameTheirMethods checks the lookups of the endpoint table: the export call, a catalog read, and an
// unknown operation, and the list of every served method.
func TestEndpointLookupsNameTheirMethods(t *testing.T) {
	if m, ok := methodOf(opExport); !ok || m != exportEndpoint.method {
		t.Fatalf("methodOf(export) = %q, %v; want the export method", m, ok)
	}
	if m, ok := methodOf(domain.OpCollectionsInfo); !ok || m != methodCollectionsInfo {
		t.Fatalf("methodOf(collections.info) = %q, %v; want %q", m, ok, methodCollectionsInfo)
	}
	if m, ok := methodOf("outline.nope"); ok || m != "" {
		t.Fatalf("methodOf(unknown) = %q, %v; want no method", m, ok)
	}
	served := Methods()
	has := func(method string) bool {
		for _, m := range served {
			if m == method {
				return true
			}
		}
		return false
	}
	if !has(exportEndpoint.method) || !has(methodCollectionsInfo) {
		t.Fatalf("Methods() = %v; want the export method and the collections method", served)
	}
}

// TestDocumentListNamesTheCollectionOnlyWhenOneIsGiven checks documentListBody: the collection filter is sent when a
// collection is named.
func TestDocumentListNamesTheCollectionOnlyWhenOneIsGiven(t *testing.T) {
	body := documentListBody(domain.NewArgs(map[string]any{"collectionId": "col-1"}))
	if body["collectionId"] != "col-1" {
		t.Fatalf("documentListBody = %v; want the collection named", body)
	}
}
