Add three independent packages to this module, split among agents:
internal/slug (turn a title into a URL slug), internal/money (format an amount
in cents as a currency string) and internal/retry (call a function again, with
backoff, until it succeeds or a limit is reached). Each comes with its own test.
None of them depends on the others.
