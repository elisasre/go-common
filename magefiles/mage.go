//go:build mage

package main

import (
	//mage:import
	_ "github.com/elisasre/mageutil/git/target"
	//mage:import
	_ "github.com/elisasre/mageutil/golangcilint/target"
	//mage:import
	_ "github.com/elisasre/mageutil/golang/target"
)
