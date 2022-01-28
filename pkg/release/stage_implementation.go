package release

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/puerco/vtrelease/pkg/env"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/release-sdk/git"
	"sigs.k8s.io/release-utils/command"
)

const versionFile = "go/vt/servenv/version.go"

type DefaultStageImplementation struct{}

func (di *DefaultStageImplementation) SetEnvironment(o *StageOptions) error {
	// Sets the environment for the next release
	e := env.New()
	e.Options.Branch = "release-12.0"
	e.Options.RepoPath = "/home/urbano/Projects/vitess/"

	nextTag, err := e.NextPatchVersion()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "getting next tag in the branc"))
	}

	// Check out the branch
	if err := e.CheckoutBranch(); err != nil {
		return errors.Wrap(err, "")
	}

	logrus.Infof("The next in the release cut will be %s", nextTag)
	return nil
}

// WriteVersionFile stamps the tag into the version.go file of the server
func (di *DefaultStageImplementation) WriteVersionFile(o *StageOptions, tag string) error {
	f, err := os.Create(versionFile)
	if err != nil {
		return errors.Wrap(err, "while opening version.go for writing")
	}
	defer f.Close()

	if _, err := fmt.Fprintf(
		f, "package servenv\n\nconst versionName = \"%s\"\n", tag[1:],
	); err != nil {
		return errors.Wrap(err, "while writing tag to version file")
	}
	return nil
}

// GenerateReleaseNotes runs the release not program to generate the changelog
func (di *DefaultStageImplementation) GenerateReleaseNotes(o *StageOptions, s *State) error {
	tmp, err := os.CreateTemp("", "release-notes-")
	if err != nil {
		return errors.Wrap(err, "creating temporary release notes file")
	}
	s.ReleaseNotesPath = tmp.Name()

	var sha_from, sha_end string
	// TODO: Get the shas
	cmd := command.NewWithWorkDir(
		o.RepoPath,      // CWD
		"release-notes", // Path to compled release notes binary
		"-from", sha_from,
		"-to,", sha_end,
		"-version", s.Version,
		"-summary", "$(SUMMARY)", /// ???
	)
	return errors.Wrap(
		cmd.RunSuccess(), "calling release notes generator",
	)
}

// GenerateJavaVersions calls Maven to generate the needed files for this veersion
func (di *DefaultStageImplementation) GenerateJavaVersions(o *StageOptions, s *State, tag string) error {
	// Invoke maven to patch the sources
	cmd := command.NewWithWorkDir(
		o.RepoPath,
		"mvn", "versions:set", fmt.Sprintf("-DnewVersion=%s", tag),
	)
	// Execute the command
	return errors.Wrapf(
		cmd.RunSuccess(), "executing maven to patch sources with tag", tag,
	)
}

// AddAndCommit adds the modified fiels and tags the repository
func (di *DefaultStageImplementation) AddAndCommit(o *StageOptions, s *State, tag string) error {
	repo, err := git.OpenRepo(o.RepoPath)
	if err != nil {
		return errors.Wrap(err, "opening repository")
	}

	// git add --all
	if err := repo.Add("--all"); err != nil {
		return errors.Wrap(err, "adding modified files to release commit")
	}

	// git commit -n -s -m "Release commit for $(RELEASE_VERSION)"
	if err := repo.UserCommit(
		fmt.Sprintf("Release commit for %s", tag),
	); err != nil {
		return errors.Wrap(err, "creating release commit")
	}
	return nil
}
