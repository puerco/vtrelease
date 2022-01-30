package commands

import (
	"github.com/puerco/vtrelease/pkg/release"
	"github.com/spf13/cobra"
)

type StageOptions struct {
	Branch       string
	GoDocVersion string
}

func AddStage(parent *cobra.Command) {
	opts := &StageOptions{}
	cmd := &cobra.Command{
		Use:           "stage",
		Short:         "Run the staging phase of the vitess release",
		Long:          "Run the staging phase of the vitess release",
		Example:       `  vtrelease stage --version=12.0.4 `,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(*cobra.Command, []string) error {
			return runStage(opts)
		},
	}

	cmd.PersistentFlags().StringVarP(
		&opts.Branch,
		"branch",
		"b",
		"",
		"branch to cut the release from. eg release-12",
	)

	cmd.PersistentFlags().StringVar(
		&opts.GoDocVersion,
		"godoc-version",
		"",
		"godoc version to tag the release commit",
	)

	parent.AddCommand(cmd)
}

func runStage(opts *StageOptions) error {
	return release.NewStage(release.StageOptions{
		RepoPath:     rootOpts.RepoPath,
		Branch:       opts.Branch,
		GoDocVersion: opts.GoDocVersion,
	}).Run()
}
