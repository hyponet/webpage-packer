package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hyponet/webpage-packer/packer"
	"os"
)

var (
	op = packer.Option{
		URL:         "",
		Output:      "output.webarchive",
		Timeout:     60,
		ClutterFree: false,
	}
)

func init() {
	flag.StringVar(&op.URL, "url", op.URL, "target url")
	flag.StringVar(&op.Output, "output", op.Output, "archive file output path")
	flag.BoolVar(&op.ClutterFree, "clutter-free", op.ClutterFree, "web page noise reduction")
}

func main() {
	flag.Parse()

	if op.URL == "" {
		fmt.Println("--url is empty")
		os.Exit(1)
	}

	if op.Output == "" {
		fmt.Println("--output is empty")
		os.Exit(1)
	}

	fmt.Printf("packing url %s\n", op.URL)
	p := packer.NewWebArchivePacker()
	err := p.Pack(context.TODO(), op)
	if err != nil {
		panic(err)
	}

	fmt.Printf("output file: %s\n", op.Output)
}
