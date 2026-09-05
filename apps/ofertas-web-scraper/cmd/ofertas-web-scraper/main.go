package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-web-scraper/internal/presentation"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ofertas-web-scraper <seed|coletar <produtoId>|run>")
		os.Exit(2)
	}
	env := presentation.LoadEnv()
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "seed":
		err = presentation.RunSeed(ctx, env)
	case "coletar":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: ofertas-web-scraper coletar <produtoId>")
			os.Exit(2)
		}
		err = presentation.RunColetar(ctx, env, os.Args[2])
	case "run":
		err = presentation.RunAll(ctx, env)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
