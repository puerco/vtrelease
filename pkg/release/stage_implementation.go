package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/puerco/vtrelease/pkg/env"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/release-sdk/git"
	"sigs.k8s.io/release-utils/command"
	"sigs.k8s.io/release-utils/util"
)

const versionFile = "go/vt/servenv/version.go"

type DefaultStageImplementation struct{}

func (di *DefaultStageImplementation) OpenRepository(o *StageOptions, s *State) error {
	repo, err := git.OpenRepo(o.RepoPath)
	if err != nil {
		return errors.Wrap(err, "opening repository")
	}
	logrus.Infof("Opened git repository in %s", o.RepoPath)
	s.Repository = repo
	return nil
}

func (di *DefaultStageImplementation) SetEnvironment(o *StageOptions, s *State) error {
	logrus.Info("💻 Setting up the environment")
	// Sets the environment for the next release
	e := env.New().WithRepository(s.Repository)

	e.Options.Branch = o.Branch

	// Check out the branch
	logrus.Infof("  > Checking out branch %s", o.Branch)
	if err := e.CheckoutBranch(); err != nil {
		return errors.Wrap(err, "")
	}

	// Add the last version cut to the tag
	prevTag, err := e.LastVersion()
	if err != nil {
		return errors.Wrap(err, "fetching the last version tag")
	}
	logrus.Infof("  > Previous release tag: %s", prevTag)
	s.PreviousVersion = prevTag

	nextTag, err := e.NextPatchVersion()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "getting next tag in the branch"))
	}
	sv, err := semver.Parse(strings.TrimPrefix(nextTag, "v"))
	if err != nil {
		return errors.Wrap(err, "parsing version tag")
	}
	logrus.Infof("  > Next release tag will be: %s", nextTag)
	s.Version = nextTag
	s.SemVer = sv

	devTag, err := e.NextDevVersion()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "getting next dev tag in the branch"))
	}
	logrus.Infof("  > Next development tag will be: %s", devTag)
	s.DevVersion = devTag

	// Record the current commit (last before the release commit)
	curCommit, err := di.GetRevSHA(o, s, "HEAD")
	if err != nil {
		return errors.Wrap(err, "trying to get the current repository commit")
	}
	logrus.Infof("  > Current branch position: %s", curCommit)
	s.CurrentCommit = curCommit

	return nil
}

// WriteVersionFile stamps the tag into the version.go file of the server
func (di *DefaultStageImplementation) WriteVersionFile(o *StageOptions, tag string) error {
	if tag == "" {
		return errors.New("unable to write version files, empty tag")
	}
	f, err := os.Create(filepath.Join(o.RepoPath, versionFile))
	if err != nil {
		return errors.Wrapf(err, "while opening %s for writing", versionFile)
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
func (di *DefaultStageImplementation) GenerateReleaseNotes(
	o *StageOptions, s *State, shaFrom, shaEnd string,
) error {
	// Ensure we have an actual range
	if shaFrom == shaEnd {
		return errors.New("start and end commits for release notes are the same")
	}
	logrus.Info("📔 Generating release notes")
	logrus.Infof("  > From SHA: %s", shaFrom)
	logrus.Infof("  > To SHA:   %s", shaEnd)

	// Record the temporary file in the in the state
	s.ReleaseNotesPath = filepath.Join(
		o.RepoPath, fmt.Sprintf(
			"doc/releasenotes/%d_%d_%d_release_notes.md",
			s.SemVer.Major, s.SemVer.Minor, s.SemVer.Patch,
		),
	)

	// The release notes scipt b0rks if the file does not
	// exist before running
	if !util.Exists(s.ReleaseNotesPath) {
		if err := os.WriteFile(s.ReleaseNotesPath, []byte{}, os.FileMode(0o644)); err != nil {
			return errors.Wrap(err, "touching release notes file")
		}
	}

	// Run the release notes generator
	cmd := command.NewWithWorkDir(
		o.RepoPath, // CWD
		"go",       // Path to compiled release notes binary
		"run",
		"./go/tools/release-notes",
		"-from", shaFrom,
		"-to", shaEnd,
		"-version", s.Version,
		// "-summary" TODO(puerco): Where to get the summary
		"-file", s.ReleaseNotesPath,
	)

	return errors.Wrap(
		cmd.RunSuccess(), "calling release notes generator",
	)
}

// GenerateJavaVersions calls Maven to generate the needed files for this veersion
func (di *DefaultStageImplementation) GenerateJavaVersions(o *StageOptions, s *State, tag string) error {
	// Invoke maven to patch the sources
	cmd := command.NewWithWorkDir(
		filepath.Join(o.RepoPath, "java"),
		"mvn", "versions:set", fmt.Sprintf("-DnewVersion=%s", tag),
	)

	// TODO(puerco): Ensure source has been patched correctly

	// Execute the command
	return errors.Wrapf(
		cmd.RunSuccess(), "executing maven to patch sources with tag %s", tag,
	)
}

func (di *DefaultStageImplementation) TagGoDocVersion(o *StageOptions, s *State) error {
	// git tag -a v$(GODOC_RELEASE_VERSION) -m "Tagging $(RELEASE_VERSION) also as $(GODOC_RELEASE_VERSION) for godoc/go modules"
	if err := s.Repository.Tag(
		s.GoDocVersion, fmt.Sprintf(
			"Tagging %s also as %s for godoc/go modules",
			s.Version, s.GoDocVersion,
		)); err != nil {
		return errors.Wrap(err, "creating godoc tag")
	}
	logrus.Infof("Tagged release commit with godoc tag %s", s.GoDocVersion)
	return nil
}

// AddAndCommit adds the modified files and tags the repository
func (di *DefaultStageImplementation) AddAndCommit(o *StageOptions, s *State, tag string) error {
	// git add --all
	if err := s.Repository.Add("--all"); err != nil {
		return errors.Wrap(err, "adding modified files to release commit")
	}

	// git commit -n -s -m "Release commit for $(RELEASE_VERSION)"
	commitMsg := fmt.Sprintf("Release commit for %s", tag)
	if strings.HasSuffix(tag, "-SNAPSHOT") {
		commitMsg = "Back to dev mode"
	}

	if err := s.Repository.UserCommit(commitMsg); err != nil {
		return errors.Wrap(err, "creating release commit")
	}
	return nil
}

// CreateTag tags the repository
func (di *DefaultStageImplementation) CreateTag(
	o *StageOptions, s *State, tag, message string,
) error {
	repo, err := git.OpenRepo(o.RepoPath)
	if err != nil {
		return errors.Wrap(err, "opening repository")
	}
	return errors.Wrapf(
		repo.Tag(tag, message),
		"tagging repo with tag %s", tag,
	)
}

func (di *DefaultStageImplementation) CheckOptions(o *StageOptions) error {
	return o.Validate()
}

// GetRevSHA ghets a git revision and returns the corresponding commit tag if found
func (di *DefaultStageImplementation) GetRevSHA(
	o *StageOptions, s *State, revision string,
) (tag string, err error) {
	commit, err := s.Repository.RevParse(revision)
	if err != nil {
		return "", errors.Wrapf(err, "getting commit for revision %s", revision)
	}
	return commit, err
}

// CheckEnvironment makes sure we are running in the environment we are supposed to
func (di *DefaultStageImplementation) CheckEnvironment(o *StageOptions) error {
	// Check that the executables we need are in the path
	logrus.Info("🔎 Looking for executables required for the build")
	for _, program := range []string{"release-notes", "mvn"} {
		path, err := exec.LookPath(program)
		if err != nil {
			return errors.Wrapf(err, "checking for %s in the system", program)
		}
		logrus.Infof("  > %s executable found in %s", program, path)
	}

	if o.Branch == "" {
		return errors.New("branch not set")
	}

	if !strings.HasPrefix(o.Branch, "release-") || !strings.HasSuffix(o.Branch, ".0") {
		return errors.New("invalid branch name")
	}

	// TODO(puerco) Check go version
	// TODO(puerco) Ensure docker config is sound

	logrus.Info("✅ Environment looks good")

	return nil
}
