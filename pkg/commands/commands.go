package commands

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
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
		"repo",
		os.Getenv("REPO_PATH"),
		"path to the vitessio/vitess repo",
	)
	cmd.PersistentFlags().BoolVar(
		&rootOpts.NoMock,
		"nomock",
		false,
		"⚠️ CAUTION",
	)

	for _, f := range []string{"repo"} {
		if err := cmd.MarkPersistentFlagRequired(f); err != nil {
			logrus.Error("marking flag as required")
		}
	}

	if err := cmd.MarkPersistentFlagDirname("repo"); err != nil {
		logrus.Error("marking command as directory")
	}
	addCommands(cmd)
	return cmd
}

func addCommands(cmd *cobra.Command) {
	AddStage(cmd)
	AddBuild(cmd)
}

func initLogging(*cobra.Command, []string) error {
	return log.SetupGlobalLogger(rootOpts.LogLevel)
}
