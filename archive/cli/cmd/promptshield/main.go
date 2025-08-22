package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/promptshield/promptshield/cmd"
	sharederrors "github.com/promptshield/promptshield/internal/shared/errors"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.Setup(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := cmd.Execute(ctx); err != nil {
		// Exit code policy: 1 for issues found (threshold), 2 for operational errors
		// Print error once to stderr
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, sharederrors.ErrFailOnThreshold) {
			os.Exit(1)
		}
		os.Exit(2)
	}
}
