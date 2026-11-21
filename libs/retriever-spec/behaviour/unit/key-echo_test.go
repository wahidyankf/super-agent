package unit_test

import (
	"testing"
)

// echoOp is a single-object read whose reply echoes its key parameter at the pointer of its id field.
const echoOp = `  - id: example-wiki.docs.fetch
    summary: Fetch one document.
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
        id:
          pointer: /id
      echo:
        param: id
        pointer: /id
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
`

// @scenario specs/libs/retriever-spec/behaviours/key-echo.feature: A single-object read that echoes its key from a named parameter is accepted
func TestSingleObjectReadThatEchoesItsKeyIsAccepted(t *testing.T) {
	requireAccepted(t, oneOp(echoOp))
}

// @scenario specs/libs/retriever-spec/behaviours/key-echo.feature: A key echo on a list read is refused at the response
func TestKeyEchoOnListReadIsRefusedAtTheResponse(t *testing.T) {
	doc := mutate(t, exampleWiki, "    response:\n      items: /items\n",
		"    response:\n      items: /items\n      echo:\n        param: cursor\n        pointer: /id\n")
	requireRefusedAt(t, doc, "/operations/0/response/echo")
}

// @scenario specs/libs/retriever-spec/behaviours/key-echo.feature: A key echo that names a parameter the operation does not declare is refused at the parameter
func TestKeyEchoNamingUndeclaredParameterIsRefusedAtTheParameter(t *testing.T) {
	doc := mutate(t, oneOp(echoOp), "        param: id", "        param: other")
	requireRefusedAt(t, doc, "/operations/0/response/echo/param")
}

// @scenario specs/libs/retriever-spec/behaviours/key-echo.feature: A key echo with no pointer to read the reply value from is refused
func TestKeyEchoWithNoPointerIsRefused(t *testing.T) {
	doc := mutate(t, oneOp(echoOp), "      echo:\n        param: id\n        pointer: /id\n", "      echo:\n        param: id\n")
	requireRefused(t, doc)
}

// @scenario specs/libs/retriever-spec/behaviours/key-echo.feature: A key echo on a write is refused
func TestKeyEchoOnWriteIsRefused(t *testing.T) {
	doc := mutate(t, oneOp(echoOp), "    access: read\n    method: GET\n", "    access: write\n    method: PUT\n")
	requireRefused(t, doc)
}
