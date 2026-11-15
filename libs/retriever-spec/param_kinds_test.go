package spec

import "testing"

// TestIconURLListAndEventListParamTypesValidate checks that a spec may name the Icon, URLList, and EventList
// parameter kinds, with exactly the validation of the operation catalog: an icon is one emoji, a named icon, or an
// attachment UUID, but never blank; an event list holds at most 50 event names, each * or dotted; a URL list holds 1
// to 10 HTTPS addresses of at most 1024 characters each. The engine binds spec arguments with the catalog rules, so
// admitting the names here is what enforces those rules there, with no new power.
func TestIconURLListAndEventListParamTypesValidate(t *testing.T) {
	doc := `version: 1
kind: example-wiki
summary: Writes with an icon, a URL list, and an event list.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.templates.create
    summary: Create a template with an icon.
    access: write
    method: POST
    path: /templates
    params:
      - name: title
        type: Name
        in: body
        required: true
      - name: icon
        type: Icon
        in: body
        required: false
    body:
      title: "{{title}}"
      icon: "{{icon}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: Weekly
          icon: star
        response:
          status: 201
          body:
            id: doc-2
        expect:
          id: doc-2
  - id: example-wiki.clients.create
    summary: Create a client with redirect addresses.
    access: write
    method: POST
    path: /clients
    params:
      - name: name
        type: Name
        in: body
        required: true
      - name: redirectUris
        type: URLList
        in: body
        required: true
    body:
      name: "{{name}}"
      redirectUris: "{{redirectUris}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          name: App
          redirectUris: ["https://app.example.test/callback"]
        response:
          status: 201
          body:
            id: doc-3
        expect:
          id: doc-3
  - id: example-wiki.hooks.create
    summary: Create a hook with event names.
    access: write
    method: POST
    path: /hooks
    params:
      - name: name
        type: Name
        in: body
        required: true
      - name: events
        type: EventList
        in: body
        required: true
    body:
      name: "{{name}}"
      events: "{{events}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          name: Hook
          events: ["documents.create"]
        response:
          status: 201
          body:
            id: doc-4
        expect:
          id: doc-4
`
	res := Validate([]byte(doc))
	if !res.Valid() {
		t.Fatalf("a spec naming Icon, URLList, and EventList is refused: %v", res.Err())
	}
}
