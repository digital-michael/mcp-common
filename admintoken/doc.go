// Package admintoken is the gate on the tools that change the user's models.
//
// The local tool server is reachable through the gateway by any key the gateway accepts, so a key alone must not be enough to
// download, delete or restart anything. A gated tool also needs the admin token: a random value kept in a file only the user
// can read, in the protected secrets folder the file tools already refuse (profiles.ProtectedDirs). Whoever can read that file
// is the user; a client that only holds a gateway key cannot. The token is checked on every call, never cached, so a token
// created or replaced later takes effect at once. With no token file the gate is closed: nothing is allowed until the user
// creates one.
package admintoken
