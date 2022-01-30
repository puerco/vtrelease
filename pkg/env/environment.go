package env

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/release-sdk/git"
)

const (
	BranchPrefix = "release-"
)

func New() *Environment {
	return &Environment{
		Options: Options{},
		impl:    &defaultImplementation{},
	}
}

//counterfeiter:generate . Implementation
type Implementation interface {
	GetRepoTags(*Options, *git.Repo) ([]string, error)
	CheckoutBranch(*Options, *git.Repo) error
}

type Environment struct {
	impl       Implementation
	Options    Options
	Repository *git.Repo
}

func (e *Environment) WithRepository(repo *git.Repo) *Environment {
	e.Repository = repo
	return e
}

func (e *Environment) SetImplementation(impl Implementation) {
	e.impl = impl
}

type Options struct {
	// Path to the vitess clone
	RepoPath string

	// Release Branch
	Branch string
}

// Validate checks if options are correct
func (o *Options) Validate() error {
	// TODO: Check branchname
	return nil
}

// BranchVersion returns the major version of the branch we
// are using, ie release-12.0 -> 12
func (e *Environment) BranchVersion() (int, error) {
	if strings.HasPrefix(e.Options.Branch, BranchPrefix) &&
		strings.HasSuffix(e.Options.Branch, ".0") {
		ver := strings.TrimSuffix(
			strings.TrimPrefix(e.Options.Branch, BranchPrefix), ".0",
		)
		i, err := strconv.Atoi(ver)
		if err != nil {
			return 0, errors.Wrap(err, "converting version to integer")
		}
		return i, nil
	}
	// TODO: check if we can cut from main
	return 0, nil
}

// NextVersion returns the next tag in the branch
func (e *Environment) NextPatchVersion() (string, error) {
	lastVer, err := e.LastVersion()
	if err != nil {
		return "", errors.Wrap(err, "while getting last version from the repo")
	}

	/// If last version is an empty string, set the 0.0 for the branch
	if lastVer == "" {
		branchVersion, err := e.BranchVersion()
		if err != nil {
			return "", errors.Wrap(err, "getting branch version")
		}
		if branchVersion == 0 {
			return "", errors.New("Unable to get major version from branch")
		}
		return fmt.Sprintf("v%d.%d.%d", branchVersion, 0, 0), nil
	}

	ver, err := semver.Parse(lastVer[1:])
	if err != nil {
		return "", errors.Wrap(err, "parsing last version tag")
	}

	return fmt.Sprintf("v%d.%d.%d", ver.Major, ver.Minor, ver.Patch+1), nil
}

// NextVersion returns the next tag in the branch
func (e *Environment) NextMinorVersion() (string, error) {
	lastVer, err := e.LastVersion()
	if err != nil {
		return "", errors.Wrap(err, "while getting last version from the repo")
	}

	/// If last version is an empty string, set the 0.0 for the branch
	if lastVer == "" {
		branchVersion, err := e.BranchVersion()
		if err != nil {
			return "", errors.Wrap(err, "getting branch version")
		}
		if branchVersion == 0 {
			return "", errors.New("Unable to get major version from branch")
		}
		return fmt.Sprintf("v%d.%d.%d", branchVersion, 0, 0), nil
	}

	ver, err := semver.Parse(lastVer[1:])
	if err != nil {
		return "", errors.Wrap(err, "parsing last version tag")
	}

	return fmt.Sprintf("v%d.%d.%d", ver.Major, ver.Minor+1, 0), nil
}

// LastVersion checks the branch for tags and returns the last cut
func (e *Environment) LastVersion() (string, error) {
	// Get the tags from the repo
	tags, err := e.impl.GetRepoTags(&e.Options, e.Repository)
	if err != nil {
		return "", errors.Wrap(err, "fetching tags from the repo")
	}

	branchVersion, err := e.BranchVersion()
	if err != nil {
		return "", errors.Wrap(err, "getting branch version")
	}
	if branchVersion == 0 {
		return "", errors.New("Unable to get major version from branch")
	}

	var greatMinor, greatPatch int
	var seen bool
	for _, tag := range tags {
		if !strings.HasPrefix(tag, fmt.Sprintf("v%d.", branchVersion)) {
			continue
		}
		seen = true
		ver, err := semver.Parse(tag[1:])
		if err != nil {
			return "", errors.Wrap(err, "parsing semantic version tag ")
		}

		if ver.Minor > uint64(greatMinor) {
			greatMinor = int(ver.Minor)
			greatPatch = int(ver.Patch)
		}
		if ver.Minor == uint64(greatMinor) {
			if ver.Patch > uint64(greatPatch) {
				greatPatch = int(ver.Patch)
			}
		}
	}

	// If there are nm tags, then its a new branch and we return 0
	if !seen {
		logrus.Warn("No tags found in the branch. Assuming new branch.")
		return "", nil
	}
	return fmt.Sprintf("v%d.%d.%d", branchVersion, greatMinor, greatPatch), nil
}

func (e *Environment) CheckoutBranch() error {
	return e.impl.CheckoutBranch(&e.Options, e.Repository)
}

type defaultImplementation struct{}

// GetRepoTags fetches the tags from the repository
func (di *defaultImplementation) GetRepoTags(o *Options, repo *git.Repo) (tags []string, err error) {
	// Checkout the branch
	if err := repo.Checkout(o.Branch); err != nil {
		return tags, errors.Wrapf(err, "checking branch %s", o.Branch)
	}

	// Search the tags to determine the next one
	return repo.Tags()
}

// CheckoutBranch checks out the branch
func (di *defaultImplementation) CheckoutBranch(o *Options, repo *git.Repo) error {
	// Checkout the branch
	if err := repo.Checkout(o.Branch); err != nil {
		return errors.Wrapf(err, "checking branch %s", o.Branch)
	}
	logrus.Infof("Checked out branch %s", o.Branch)
	return nil
}
