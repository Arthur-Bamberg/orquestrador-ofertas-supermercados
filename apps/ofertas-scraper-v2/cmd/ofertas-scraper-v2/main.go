package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/presentation"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ofertas-scraper-v2 <discover|run>")
		os.Exit(2)
	}
	env := presentation.LoadEnv()
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "discover":
		err = presentation.RunDiscover(ctx, env)
	case "run":
		err = presentation.RunDownload(ctx, env)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
