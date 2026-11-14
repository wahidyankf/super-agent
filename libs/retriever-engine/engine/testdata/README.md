# testdata

[wiki.yml](./wiki.yml) is the connector spec that the engine tests read. It describes an example wiki on the host
`docs.example.test`, and it names only example hosts and example variables (`EXAMPLE_WIKI_API_KEY`,
`EXAMPLE_WIKI_EMAIL`). It is never a real source.

The fixture covers these cases:

- `example-wiki.docs.list`: a cursor list with a constant header, a default page size, and a page cap.
- `example-wiki.docs.page`: an offset list with a default page size and offset.
- `example-wiki.docs.search`: a query value that also fills a header.
- `example-wiki.docs.get`: a path slot, and a Markdown converter that the engine refuses in this pull request.
- `example-wiki.docs.create`: a write, which the engine refuses before any request.
- `example-wiki.collections.list`: a list with no paging, and the `list-collections` role.

The tests that need the collection tree and the document read add their own operations to this fixture, in the test
files, and the fixture itself keeps only the operations above.
