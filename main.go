package main

import "github.com/tilotech/batch-graphql/cmd"

var version = ""

func main() {
	cmd.Version = version
	cmd.Execute()
}
