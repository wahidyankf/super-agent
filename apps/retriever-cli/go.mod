module github.com/wahidyankf/super-agent/apps/retriever-cli

go 1.27.0

toolchain go1.27.4

require (
	github.com/99designs/gqlgen v0.17.98
	github.com/cucumber/godog v0.16.0
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/vektah/gqlparser/v2 v2.5.64
	github.com/wahidyankf/super-agent/libs/retriever-connector-jira v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-connector-outline v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-contract v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-engine v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-runtime v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-spec v0.4.0
	github.com/wahidyankf/super-agent/libs/retriever-testkit v0.4.0
)

require (
	github.com/agnivade/levenshtein v1.2.2 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/cucumber/gherkin/go/v42 v42.0.0 // indirect
	github.com/cucumber/messages/go/v34 v34.2.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.5 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sosodev/duration v1.4.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/urfave/cli/v3 v3.14.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/telemetry v0.0.0-20261010193011-f71dc92da45e // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	golang.org/x/vuln v1.9.0 // indirect
)

tool (
	github.com/99designs/gqlgen
	golang.org/x/vuln/cmd/govulncheck
)

replace (
	github.com/wahidyankf/super-agent/libs/retriever-connector-jira => ../../libs/retriever-connector-jira
	github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi => ../../libs/retriever-connector-kiwi
	github.com/wahidyankf/super-agent/libs/retriever-connector-outline => ../../libs/retriever-connector-outline
	github.com/wahidyankf/super-agent/libs/retriever-contract => ../../libs/retriever-contract
	github.com/wahidyankf/super-agent/libs/retriever-engine => ../../libs/retriever-engine
	github.com/wahidyankf/super-agent/libs/retriever-runtime => ../../libs/retriever-runtime
	github.com/wahidyankf/super-agent/libs/retriever-spec => ../../libs/retriever-spec
	github.com/wahidyankf/super-agent/libs/retriever-testkit => ../../libs/retriever-testkit
)
