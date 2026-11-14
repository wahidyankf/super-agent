module github.com/wahidyankf/super-agent/libs/retriever-engine

go 1.27.0

toolchain go1.27.4

require (
	github.com/wahidyankf/super-agent/libs/retriever-contract v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-spec v0.4.0
)

require (
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace (
	github.com/wahidyankf/super-agent/libs/retriever-contract => ../retriever-contract
	github.com/wahidyankf/super-agent/libs/retriever-spec => ../retriever-spec
)
