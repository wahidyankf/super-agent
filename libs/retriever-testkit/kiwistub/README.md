# Kiwi stub

A scripted Kiwi TCMS server. It answers `POST /json-rpc/` with JSON-RPC 2.0 replies in the envelope and the error shapes
that the Kiwi adapter reads. The dataset comes from the test. The stub checks the Bearer key and keeps only a SHA-256
digest of each Authorization value, never the value itself.

The test can script one fault for every call, for one method, or for the next call only, and it can set a JSON-RPC error
for a method.
