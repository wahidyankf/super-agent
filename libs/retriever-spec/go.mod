module github.com/wahidyankf/super-agent/libs/retriever-spec

go 1.27.0

toolchain go1.27.4

require (
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	github.com/wahidyankf/super-agent/libs/retriever-contract v0.4.0
	go.yaml.in/yaml/v3 v3.0.6
)

require golang.org/x/text v0.43.0 // indirect

replace github.com/wahidyankf/super-agent/libs/retriever-contract => ../retriever-contract
