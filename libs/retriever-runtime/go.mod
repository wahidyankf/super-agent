module github.com/wahidyankf/super-agent/libs/retriever-runtime

go 1.27.0

toolchain go1.27.4

require (
	github.com/wahidyankf/super-agent/libs/retriever-contract v0.3.0
	go.yaml.in/yaml/v3 v3.0.5
)

replace github.com/wahidyankf/super-agent/libs/retriever-contract => ../retriever-contract
