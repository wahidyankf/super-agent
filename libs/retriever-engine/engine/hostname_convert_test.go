package engine_test

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// hostnameYAML is a read whose reply holds addresses, mapped with the hostname
// converter to the host of each address.
const hostnameYAML = `version: 1
kind: example-wiki
summary: Reads with host-only address fields of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.hooks.list
    summary: List the webhook subscriptions of the wiki.
    access: read
    method: GET
    path: /hooks
    response:
      items: /items
      fields:
        id:
          pointer: /id
          required: true
          format: id
        urlHost:
          pointer: /url
          convert: hostname
        settings:
          object:
            instanceUrl:
              pointer: /settings/instanceUrl
              convert: hostname
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: h1
                url: https://hooks.example.test/x
        expect:
          id: h1
`

// TestAHostnameFieldMapsTheHostOfAnAddress checks the hostname converter: the
// output field holds the host name of the reply address, without the path,
// the query, the port, or the user name.
func TestAHostnameFieldMapsTheHostOfAnAddress(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, hostnameYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1","url":"https://user@hooks.example.test:8443/synthetic/path?token=abc","settings":{"instanceUrl":"https://instance.example.test/app"}}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	item := rec.Items()[0]
	if got, _ := item.Get("urlHost"); got != "hooks.example.test" {
		t.Fatalf("urlHost = %#v, want hooks.example.test", got)
	}
	settings, ok := item.Get("settings")
	if !ok {
		t.Fatal("settings is missing")
	}
	settingsRec, ok := settings.(domain.Record)
	if !ok {
		t.Fatalf("settings = %#v, want a record", settings)
	}
	if got, _ := settingsRec.Get("instanceUrl"); got != "instance.example.test" {
		t.Fatalf("settings.instanceUrl = %#v, want instance.example.test", got)
	}
}

// TestAHostnameFieldMapsAnUnusableAddressToEmpty checks that an address the
// converter cannot parse maps to the empty text, so a broken address never
// reaches the answer.
func TestAHostnameFieldMapsAnUnusableAddressToEmpty(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, hostnameYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1","url":"://bad"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got, _ := rec.Items()[0].Get("urlHost"); got != "" {
		t.Fatalf("urlHost = %#v, want empty", got)
	}
}

// TestAHostnameFieldMissingMapsToEmpty checks that a missing address maps to
// the empty text, as the Go connectors read a missing address.
func TestAHostnameFieldMissingMapsToEmpty(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, hostnameYAML))
	stub.Script("GET", "/v1/hooks", page(`{"items":[{"id":"h1"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.hooks.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got, ok := rec.Items()[0].Get("urlHost"); !ok || got != "" {
		t.Fatalf("urlHost = (%#v, %v), want (\"\", true)", got, ok)
	}
}
