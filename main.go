/*
Copyright © 2024 Oussama Abboud oussama.abboud@avisto.com
*/
package main

import "vresq/cmd"

var (
	version = "dev"
)

func main() {
	cmd.SetVersionInfo(version)
	cmd.Execute()
}
