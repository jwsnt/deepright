package main

import (
	"flag"
	"fmt"
	"os"

	"site/skillrepairprompt"
)

func main() {
	var path string
	flag.StringVar(&path, "path", "", "absolute path to the invalid SKILL.md file")
	flag.Parse()
	if path == "" && flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	prompt, err := skillrepairprompt.Build(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(prompt)
}
