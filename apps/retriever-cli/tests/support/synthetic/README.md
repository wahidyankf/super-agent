# Synthetic Outline replies

One JSON reply for each Outline method that the write and admin commands call, named `<method>.json`. The behaviour
harness reads these files, embedded through the `synthetic` package, and gives each one to the Outline stub as a fixed
reply. The stub answers eight methods from its dataset instead, so the stub never uses the replies for these methods:
`auth.info`, `collections.list`, `collections.info`, `collections.documents`, `documents.info`, `documents.export`,
`documents.list`, and `documents.search`.

All values are invented: the hosts use the `example.test` name and the identifiers are synthetic.
