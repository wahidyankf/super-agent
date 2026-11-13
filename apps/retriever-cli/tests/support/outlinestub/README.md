# Outline stub

A scripted stand-in for the Outline API. It answers the methods the adapter calls, from a dataset the test sets and
from the replay fixtures in [testdata/](testdata/README.md). It records every request: the method, the HTTP method,
the path, the host, the Authorization header, and the body.

The test can script faults for one method or for the first N calls, set a fixed reply for a method, and set the
signed-in account. A method that the stub does not offer answers 404 "Endpoint not found", as the real router does for
an unknown method.
