package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/presentation"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ofertas-scraper <run|seed>")
		os.Exit(2)
	}
	env := presentation.LoadEnv()
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "run":
		err = presentation.RunDaily(ctx, env)
	case "seed":
		err = presentation.RunSeed(ctx, env)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
