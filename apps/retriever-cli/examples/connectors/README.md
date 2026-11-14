# Connector spec examples

`example-wiki.yml` is a connector spec for the example wiki. Its endpoint is `example.test`, and its key comes from the
variable `EXAMPLE_WIKI_API_KEY`. It has one read operation, `example-wiki docs list`, and two write operations, create
and delete. The cmd integration tests run the list operation over loopback HTTPS against the replay stub.

Copy the file into a connectors folder, or pass that folder with `--connectors-dir`. The
[add a spec connector](../../../../docs/how-to/add-a-spec-connector.md) how-to explains each step. The
`cmd/retriever` integration tests load this file, so an edit that breaks it fails the build.
