package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"site/filenamelookup"
)

func main() {
	var root string
	var name string
	flag.StringVar(&root, "root", "", "absolute workspace root")
	flag.StringVar(&name, "name", "", "selected file name")
	flag.Parse()
	if root == "" || name == "" {
		fmt.Fprintln(os.Stderr, "root and name are required")
		os.Exit(1)
	}
	matches, err := filenamelookup.Lookup(root, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(matches); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
