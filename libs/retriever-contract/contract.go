// Package contract is the root of the retriever contract library. It holds the version of the contract that the
// connectors and the app build against. The types live in the domain, ports and operations packages.
package contract

// ContractVersion is the version of the contract. The libraries are released in lockstep, so the value matches the
// release tag of the libraries (libs/<name>/v0.4.0 and the binary tag). Version 0.4.0 opens the kind set: a composition
// root registers kinds and rows at start-up.
const ContractVersion = "0.4.0"
