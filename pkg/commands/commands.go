package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/release-utils/log"
)

type rootOptions struct {
	LogLevel string
	RepoPath string
	NoMock   bool
}

var rootOpts = &rootOptions{}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "vtrelease",
		Short:             "Vitess release process controller",
		PersistentPreRunE: initLogging,
	}

	cmd.PersistentFlags().StringVar(
		&rootOpts.LogLevel,
		"log-level",
		"info",
		fmt.Sprintf("the logging verbosity, either %s", log.LevelNames()),
	)

	cmd.PersistentFlags().StringVar(
		&rootOpts.RepoPath,
		"repopath",
		os.Getenv("REPO_PATH"),
		"path to the vitessio/vitess repo",
	)
	cmd.PersistentFlags().BoolVar(
		&rootOpts.NoMock,
		"nomock",
		false,
		"⚠️ CAUTION",
	)
	addCommands(cmd)
	return cmd
}

func addCommands(cmd *cobra.Command) {
	AddStage(cmd)
}

func initLogging(*cobra.Command, []string) error {
	return log.SetupGlobalLogger(rootOpts.LogLevel)
}