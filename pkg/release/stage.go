package release

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/release-sdk/git"
)

type StageImplementation interface {
	CheckOptions(*StageOptions) error
	SetEnvironment(*StageOptions, *State) error
	OpenRepository(*StageOptions, *State) error
	GenerateReleaseNotes(*StageOptions, *State) error
	TagRepository(*StageOptions, *State) error
	WriteVersionFile(*StageOptions, string) error
	GenerateJavaVersions(*StageOptions, *State, string) error
	AddAndCommit(*StageOptions, *State, string) error
	CreateTag(*StageOptions, *State, string) error
	TagGoDocVersion(o *StageOptions, s *State) error
}

type StageOptions struct {
	RepoPath string
}

func (o *StageOptions) Validate() error {
	// TODO: Implement
	return nil
}

type State struct {
	Version          string
	DevVersion       string
	GoDocVersion     string
	ReleaseNotesPath string
	Repository       *git.Repo
}

type Stage struct {
	Options StageOptions
	impl    StageImplementation
	State   State
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
	if err := s.impl.CheckOptions(&s.Options); err != nil {
		return errors.Wrap(err, "checking staging options")
	}
	if err := s.impl.OpenRepository(&s.Options, &s.State); err != nil {
		return errors.Wrap(err, "opening repository")
	}
	// FIXME: Check env . Check java compiler
	return s.impl.SetEnvironment(&s.Options, &s.State)
}

func (s *Stage) GenerateReleaseNotes() error {
	return s.impl.GenerateReleaseNotes(&s.Options, &s.State)
}

// Write the version file and tag the repo. Each for the release and dev
// versions.
func (s *Stage) TagRepository() error {
	// We cycle here the two release versions
	for _, tag := range []string{s.State.Version, s.State.DevVersion} {
		if err := s.impl.GenerateJavaVersions(&s.Options, &s.State, tag); err != nil {
			return errors.Wrapf(err, "writing generating version %s in java")
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
		if err := s.impl.CreateTag(&s.Options, &s.State, tag); err != nil {
			return errors.Wrap(err, "creating tag")
		}
	}
	return nil
}
