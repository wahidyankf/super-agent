# Jira stub

A scripted stand-in for Jira Cloud. It answers the REST paths that retriever reads, from a synthetic dataset, and it
applies scripted faults per path. It records each request, so a test can assert the calls the client made. It accepts
only the synthetic credentials the test sets, and the request log never holds the token.
