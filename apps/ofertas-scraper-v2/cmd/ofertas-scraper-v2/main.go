package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper-v2/internal/presentation"
)

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	env := presentation.LoadEnv()
	ctx := context.Background()
	var err error
	switch cmd {
	case "seed":
		err = presentation.RunSeed(ctx, env)
	case "coletar":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: ofertas-scraper-v2 coletar <termo>")
			os.Exit(2)
		}
		err = presentation.RunColetar(ctx, env, os.Args[2])
	case "serve":
		err = presentation.RunServe(env)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
