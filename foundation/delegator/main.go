package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	var kubeconfig string
	flag.StringVar(&kubeconfig, "regionalkubeconfig", "", "optional path to kubeconfig file")
	flag.Parse()
	if err := Run(context.Background(), kubeconfig); err != nil {
		fmt.Fprintf(os.Stderr, "delegator failed: %v\n", err)
		os.Exit(1)
	}
}
