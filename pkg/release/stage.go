package release

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/release-sdk/git"
)

type StageImplementation interface {
	CheckOptions(*StageOptions) error
	SetEnvironment(*StageOptions, *State) error
	OpenRepository(*StageOptions, *State) error
	GenerateReleaseNotes(*StageOptions, *State, string, string) error
	WriteVersionFile(*StageOptions, string) error
	GenerateJavaVersions(*StageOptions, *State, string) error
	AddAndCommit(*StageOptions, *State, string) error
	CreateTag(*StageOptions, *State, string, string) error
	TagGoDocVersion(o *StageOptions, s *State) error
	GetRevSHA(*StageOptions, *State, string) (string, error)
	CheckEnvironment(*StageOptions) error
}

type StageOptions struct {
	// RepoPath is where the vitess repository is located
	RepoPath string

	// Branch is the branch from which we will release. Eg release-12.0
	Branch string

	// GoDocVersion
	GoDocVersion string
}

func (o *StageOptions) Validate() error {
	// TODO: Implement
	if o.RepoPath == "" {
		return errors.New("Path to repository not defined")
	}

	return nil
}

type State struct {
	// The tag we will cut
	Version string

	//
	DevVersion string

	// PreviousVersion cotains the last tag that was cut
	PreviousVersion string

	// GoDoc tag to apply to the release commit in addition to the release tag
	GoDocVersion string

	// Path to store the release notes file
	ReleaseNotesPath string

	// Current commit contains the last commit in the release before we add the release commit
	CurrentCommit string

	// SHA of the commit that will contain the tag
	ReleasePoint string

	Repository *git.Repo
}

type Stage struct {
	Options StageOptions
	impl    StageImplementation
	State   State
}

func NewStage(o StageOptions) *Stage {
	return &Stage{
		impl:    &DefaultStageImplementation{},
		Options: o,
	}
}

// Run executes the release run
func (s *Stage) Run() error {
	if err := s.PrepareEnvironment(); err != nil {
		return errors.Wrap(err, "setting up environment")
	}

	if err := s.GenerateReleaseNotes(); err != nil {
		return errors.Wrap(err, "generating release notes")
	}

	if err := s.TagRepository(); err != nil {
		return errors.Wrap(err, "tagging repo")
	}
	return nil
}

func (s *Stage) PrepareEnvironment() error {
	// Verify the runner environment
	if err := s.impl.CheckEnvironment(&s.Options); err != nil {
		return errors.Wrap(err, "checking build environment")
	}

	// Check all options are valid
	if err := s.impl.CheckOptions(&s.Options); err != nil {
		return errors.Wrap(err, "checking staging options")
	}

	// Open the repository
	if err := s.impl.OpenRepository(&s.Options, &s.State); err != nil {
		return errors.Wrap(err, "opening repository")
	}

	// Set required environment values
	return s.impl.SetEnvironment(&s.Options, &s.State)
}

func (s *Stage) GenerateReleaseNotes() error {
	// Get the commit sha of the previous release
	fromSha, err := s.impl.GetRevSHA(&s.Options, &s.State, s.State.PreviousVersion)
	if err != nil {
		return errors.Wrap(err, "getting previous release commit sha")
	}

	// Current commit is the tag commit. Therefore, we will generate the
	// release notes up to the previous one
	toSha, err := s.impl.GetRevSHA(
		&s.Options, &s.State, fmt.Sprintf("%s~1", s.State.CurrentCommit),
	)
	if err != nil {
		return errors.Wrap(err, "getting previous release commit sha")
	}

	// Run the release notes generator
	return s.impl.GenerateReleaseNotes(&s.Options, &s.State, fromSha, toSha)
}

// Write the version file and tag the repo. Each for the release and dev
// versions.
func (s *Stage) TagRepository() error {
	// We cycle here the two release versions
	for _, tag := range []string{s.State.Version, s.State.DevVersion} {
		if err := s.impl.GenerateJavaVersions(&s.Options, &s.State, tag); err != nil {
			return errors.Wrapf(err, "generating version %s files in java source", s.State.Version)
		}
		// Write the version file
		if err := s.impl.WriteVersionFile(&s.Options, tag); err != nil {
			return errors.Wrapf(err, "writing tag %s to code", tag)
		}

		if tag == s.State.DevVersion {
			continue
		}

		// If we have a GO_DOC
		if s.State.GoDocVersion != "" {
			if err := s.impl.TagGoDocVersion(&s.Options, &s.State); err != nil {
				return errors.Wrap(err, "tagging godoc version")
			}
		}

		if err := s.impl.AddAndCommit(&s.Options, &s.State, tag); err != nil {
			return errors.Wrap(err, "creating tag commit")
		}

		// git tag -m Version\ $(RELEASE_VERSION) v$(RELEASE_VERSION)
		if err := s.impl.CreateTag(&s.Options, &s.State, tag, fmt.Sprintf("Release commit for %s", tag)); err != nil {
			return errors.Wrap(err, "creating tag")
		}
	}
	return nil
}
