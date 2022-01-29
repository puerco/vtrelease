package commands

import (
	"github.com/puerco/vtrelease/pkg/release"
	"github.com/spf13/cobra"
)

func AddStage(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:           "stage",
		Short:         "Run the staging phase of the vitess release",
		Long:          "Run the staging phase of the vitess release",
		Example:       `  vtrelease stage --version=12.0.4 `,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(*cobra.Command, []string) error {
			return runStage()
		},
	}
	parent.AddCommand(cmd)
}

func runStage() error {
	return release.NewStage(release.StageOptions{
		RepoPath: rootOpts.RepoPath,
	}).Run()
}
