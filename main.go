package main

import (
	"gabor-boros/sprint-update/cmd"
)

var version string
var commit string
var date string

func main() {
	cmd.Execute(version, commit, date)
}
