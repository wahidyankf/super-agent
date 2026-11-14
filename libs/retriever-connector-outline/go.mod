module github.com/wahidyankf/super-agent/libs/retriever-connector-outline

go 1.27.0

toolchain go1.27.4

require (
	github.com/wahidyankf/super-agent/libs/retriever-contract v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-testkit v0.4.0
)

replace (
	github.com/wahidyankf/super-agent/libs/retriever-contract => ../retriever-contract
	github.com/wahidyankf/super-agent/libs/retriever-testkit => ../retriever-testkit
)
