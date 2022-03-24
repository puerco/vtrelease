package commands

import (
	"errors"
	"os"

	"github.com/puerco/vtrelease/pkg/release"
	"github.com/spf13/cobra"
)

type BuildOptions struct {
	Branch  string
	Version string
}

func AddBuild(parent *cobra.Command) {
	opts := &BuildOptions{}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build command set",
		// Long:          "Run the staging phase of the vitess release",
		// Example:       `  vtrelease build --version=12.0.4 `,
		//SilenceUsage:  true,
		SilenceErrors: true,
	}

	image := &cobra.Command{
		Use:   "image --version=vM.m.p IMAGE_NAME",
		Short: "Build vitess container images",
		// Long:          "Run the staging phase of the vitess release",
		// Example:       `  vtrelease build --version=12.0.4 `,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("you must soecify the name if the image to build")
			}
			if len(args) > 1 {
				return errors.New("image should only process one image a a time ")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImageBuild(opts, args[0])
		},
	}

	image.PersistentFlags().StringVar(
		&opts.Version,
		"version",
		os.Getenv("VT_BASE_VER"),
		"version tag to build",
	)

	image.PersistentFlags().StringVar(
		&opts.Version,
		"staging-registry",
		release.DefaultBuildOptions.StagingRegistry,
		"registry where images are staged",
	)

	image.PersistentFlags().StringVarP(
		&opts.Branch,
		"branch",
		"b",
		"",
		"branch to cut the release from. eg release-12",
	)

	cmd.AddCommand(image)
	parent.AddCommand(cmd)
}

func runImageBuild(opts *BuildOptions, image string) error {
	o := release.DefaultBuildOptions
	o.VTBaseVersion = opts.Version
	o.RepoPath = rootOpts.RepoPath

	return release.NewBuild(o).Image(image)
}
