// Command pin prints the digests a pinned.Asset needs for one archive: the
// archive's sha256, and the sha256 of each named file inside it.
//
//	go run github.com/meop/golie/cmd/pin -url <archive url> mallet wedge
//
// It is how a pin is bumped: run it for each platform's asset of the new
// release and paste what it prints into the tool's digest table.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/meop/golie/pinned"
)

func main() {
	url := flag.String("url", "", "the archive to read")
	flag.Parse()
	if *url == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: pin -url <archive url> <file>...")
		os.Exit(2)
	}
	if err := run(*url, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(url string, files []string) error {
	asset, err := pinned.Read(context.Background(), url, files)
	if err != nil {
		return err
	}
	fmt.Printf("%q: {SHA256: %q, Files: map[string]string{", filepath.Base(url), asset.SHA256)
	for i, name := range files {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%q: %q", name, asset.Files[name])
	}
	fmt.Println("}},")
	return nil
}
