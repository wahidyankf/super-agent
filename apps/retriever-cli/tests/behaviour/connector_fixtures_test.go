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

// wikiPostReadOperation is a read sent over POST, with the marker. Its body holds one declared slot, and its example
// answers the request. The text of the query is the only value the body carries.
const wikiPostReadOperation = `  - id: example-wiki.docs.search
    summary: Search documents by text.
    access: read
    method: POST
    read_over_post: true
    path: /docs/search
    params:
      - name: query
        type: Query
        in: body
        required: true
    body:
      text: "{{query}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          query: draft
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// specWithPostRead is the example spec with one more operation, a read over POST.
func specWithPostRead() string {
	return wikiSpec + wikiPostReadOperation
}

// checkSpec is the wiki spec the conformance scenarios check: one list
// operation with one example. Its endpoint suffix and names are synthetic, and
// its key is named by a variable only.
const checkSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request: {}
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

// specWithNoExamples is the check spec with the examples of its operation
// removed, so validate refuses it naming the operation.
func specWithNoExamples() string {
	end := strings.Index(checkSpec, "    examples:\n")
	if end < 0 {
		panic("the check spec has no examples block to remove")
	}
	return checkSpec[:end]
}

// specWithUnproducedField is the check spec whose example expects a declared
// field the canned reply does not produce, so test refuses it naming the
// operation, the example, and the field.
func specWithUnproducedField() string {
	return strings.Replace(checkSpec,
		"            items:\n              - id: doc-1\n                title: First document\n",
		"            items:\n              - id: doc-1\n", 1)
}

// specWithOutlineKeyAndPlantedCredential is the check spec with the Outline
// key variable and the secret planted in the canned reply and the expectation.
// The built binary reads only the built-in credential variables, so the
// end-to-end level needs the Outline variable to see the secret at all.
func specWithOutlineKeyAndPlantedCredential(secret string) string {
	doc := strings.Replace(checkSpec, "  credential_env: EXAMPLE_WIKI_API_KEY",
		"  credential_env: EXAMPLE_OUTLINE_API_KEY", 1)
	return strings.ReplaceAll(doc, "First document", secret)
}
