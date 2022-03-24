package release

import "github.com/pkg/errors"

type Build struct {
	Options BuildOptions
	impl    BuildImplementation
	State   State
}

func NewBuild(o BuildOptions) *Build {
	return &Build{
		impl:    &defaultBuildImplementation{},
		Options: o,
		State:   State{},
	}
}

type BuildOptions struct {
	// Path to vitess repository
	RepoPath string

	VTBaseVersion        string   // TODO(puerc): Move to
	DebianVersions       []string //
	DefaultDebianVersion string   // buster

	// Registry where images are staged
	StagingRegistry string
}

var DefaultBuildOptions = BuildOptions{
	DebianVersions:       []string{"buster", "bullseye"},
	DefaultDebianVersion: "buster",
	StagingRegistry:      "gcr.io/puerco-chainguard/vitess/staging",
}

func (o *BuildOptions) Validate() error {
	return nil
}

func (b *Build) Image(image string) error {
	if err := b.impl.ValidateImageOpts(&b.Options, &b.State, image); err != nil {
		return errors.Wrap(err, "validating image build options")
	}
	return b.impl.BuildImage(&b.Options, &b.State, image)

}
