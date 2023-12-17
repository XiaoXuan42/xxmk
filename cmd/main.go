package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

func main() {
	var opts struct {
		Path string `short:"p" long:"path" description:"input filename" required:"true"`
	}
	_, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}
	content, err := os.ReadFile(opts.Path)
	if err != nil {
		log.Fatalln("File not found: ", opts.Path)
	}
	fmt.Println(string(content))
}