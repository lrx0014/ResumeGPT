package migrations

import "embed"

// Files contains the immutable SQL migration sources.
//
//go:embed *.sql
var Files embed.FS
