//go:build mage

package main

import (
	"context"

	"github.com/elisasre/mageutil"
)

// UnitTest whole repo
func UnitTest(ctx context.Context) error {
	return mageutil.UnitTest(ctx)
}

// Lint all go files.
func Lint(ctx context.Context) error {
	return mageutil.LintAll(ctx)
}

// Clean removes all files ignored by git
func Clean(ctx context.Context) error {
	return mageutil.Clean(ctx)
}
