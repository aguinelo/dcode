//go:build !unix

package app

// secureDir has nothing to check here: there is no uid and no mode bits, and
// access lives in ACLs this does not interpret. The server makes the directory
// as it makes any other. No sandbox backend exists for these platforms yet, so
// a session here runs unconfined, in full access, whatever the socket's place.
func secureDir(string) error { return nil }
