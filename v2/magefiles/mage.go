//go:build mage

package main

import (
	//mage:import
	_ "github.com/elisasre/mageutil/git/target"
	//mage:import go
	_ "github.com/elisasre/mageutil/tool/golangcilint"
	//mage:import
	_ "github.com/elisasre/mageutil/golang/target"
)
