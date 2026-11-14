package behaviour

import "strings"

// wikiSpec is the example wiki spec of the connector behaviours: one list operation with one example. Its endpoint suffix
// and its names are synthetic, and its key is named by a variable only. Its kind is example-wiki.
const wikiSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request:
          limit: 20
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
`

// wikiConverterOperation is a read whose reply needs a Markdown converter. The engine refuses it before any request.
const wikiConverterOperation = `  - id: example-wiki.docs.get
    summary: Get one document as Markdown.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        body:
          pointer: /body
          convert: markdown-from-html
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            body: "<p>Hello</p>"
        expect:
          body: Hello
`

// specCLIOffMCPOn is the example spec with the command line switched off for its list operation, and its MCP tool kept.
func specCLIOffMCPOn() string {
	return strings.Replace(wikiSpec, "    access: read\n", "    access: read\n    cli: false\n    mcp: true\n", 1)
}

// specCLIOnMCPOff is the example spec with its MCP tool switched off, and its command kept.
func specCLIOnMCPOff() string {
	return strings.Replace(wikiSpec, "    access: read\n", "    access: read\n    cli: true\n    mcp: false\n", 1)
}

// specWithAuthHeader is the example spec with an Authorization header on its list operation, which a spec may not set. The
// header value is synthetic, and a finding must never print it.
func specWithAuthHeader() string {
	return strings.Replace(wikiSpec, "    path: /docs\n",
		"    path: /docs\n    headers:\n      Authorization: Bearer example-token-value\n", 1)
}

// specWithKind is the example spec with its kind, and the kind of each operation, replaced by kind.
func specWithKind(kind string) string {
	return strings.ReplaceAll(wikiSpec, "example-wiki", kind)
}

// specWithConverter is the example spec with one more operation, whose reply needs a Markdown converter.
func specWithConverter() string {
	return wikiSpec + wikiConverterOperation
}
