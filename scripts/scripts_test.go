package main

import "testing"

// TestNop allows go test ./scripts to build when no script build tag is set.
// Run scripts with: go run -tags gen_sample . or go run -tags convert_local .
func TestNop(t *testing.T) {
	t.Skip("scripts are entrypoints; run with -tags gen_sample or -tags convert_local")
}
