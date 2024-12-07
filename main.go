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
		FilePath:    "output.webarchive",
		Timeout:     60,
		ClutterFree: false,
	}
	packType = "webarchive"
)

func init() {
	flag.StringVar(&op.URL, "url", op.URL, "target url")
	flag.StringVar(&op.FilePath, "output", op.FilePath, "archive file output path")
	flag.StringVar(&packType, "pack-type", packType, "archive file type: webarchive html")
	flag.BoolVar(&op.ClutterFree, "clutter-free", op.ClutterFree, "web page noise reduction")
}

func main() {
	flag.Parse()

	if op.URL == "" {
		fmt.Println("--url is empty")
		os.Exit(1)
	}

	if op.FilePath == "" {
		fmt.Println("--output is empty")
		os.Exit(1)
	}

	op.EnablePrivateNet = true
	fmt.Printf("packing url %s\n", op.URL)

	var p packer.Packer
	switch packType {
	case "webarchive":
		p = packer.NewWebArchivePacker()
	case "html":
		p = packer.NewHtmlPacker()
	default:
		fmt.Printf("unknown pack type %s\n", packType)
		os.Exit(1)
	}

	err := p.Pack(context.TODO(), op)
	if err != nil {
		panic(err)
	}

	fmt.Printf("output file: %s\n", op.FilePath)
}
