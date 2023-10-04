//go:build mage

package main

import (
	"context"
	"os"

	"github.com/elisasre/mageutil"
)

const (
	AppName   = "kaas-kubectl-proxy"
	RepoURL   = "https://github.com/elisasre/kaas-kubectl-proxy"
	ImageName = "quay.io/elisaoyj/sre-kaas-kubectl-proxy"
)

// Build binaries for executables under ./cmd
func Build(ctx context.Context) error {
	return mageutil.BuildAll(ctx)
}

// UnitTest whole repo
func UnitTest(ctx context.Context) error {
	return mageutil.UnitTest(ctx)
}

// IntegrationTest whole repo
func IntegrationTest(ctx context.Context) error {
	return mageutil.IntegrationTest(ctx, "./cmd/"+AppName)
}

// Run binary for kaas-kubectl-proxy
func Run(ctx context.Context) error {
	return mageutil.Run(ctx, AppName)
}

// Lint all go files.
func Lint(ctx context.Context) error {
	return mageutil.LintAll(ctx)
}

// VulnCheck all go files.
func VulnCheck(ctx context.Context) error {
	return mageutil.VulnCheckAll(ctx)
}

// LicenseCheck all files.
func LicenseCheck(ctx context.Context) error {
	return mageutil.LicenseCheck(ctx, os.Stdout, mageutil.CmdDir+AppName)
}

// Image creates docker image.
func Image(ctx context.Context) error {
	return mageutil.DockerBuildDefault(ctx, ImageName, RepoURL)
}

// PushImage creates docker image.
func PushImage(ctx context.Context) error {
	return mageutil.DockerPushAllTags(ctx, ImageName)
}

// Clean removes all files ignored by git
func Clean(ctx context.Context) error {
	return mageutil.Clean(ctx)
}
