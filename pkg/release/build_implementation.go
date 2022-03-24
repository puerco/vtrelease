package release

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/release-utils/command"
)

type BuildImplementation interface {
	BuildImage(*BuildOptions, *State, string) error
	ValidateImageOpts(*BuildOptions, *State, string) error
}

type defaultBuildImplementation struct {
}

func (di *defaultBuildImplementation) ValidateImageOpts(o *BuildOptions, s *State, name string) error {
	return nil
}

func (di *defaultBuildImplementation) BuildImage(o *BuildOptions, s *State, imageName string) error {
	// echo "####### Building vitess/vt:$debian_version"

	// Validate the image name by checking a dir in docker/k8s/${name}

	// TODO(puerco): Only push second tag when default debian version
	for _, distro := range o.DebianVersions {
		err := command.NewWithWorkDir(
			filepath.Join(o.RepoPath, "docker/k8s"),
			"docker", "buildx", "build",
			"--build-arg", fmt.Sprintf("VT_BASE_VER=%s", o.VTBaseVersion),
			"--build-arg", fmt.Sprintf("DEBIAN_VER=%s-slim", distro),
			"--tag", fmt.Sprintf("%s/%s:%s-%s", o.StagingRegistry, imageName, o.VTBaseVersion, distro),
			"--tag", fmt.Sprintf("%s/%s:%s", o.StagingRegistry, imageName, o.VTBaseVersion),
			"--output", "type=image,push=true",
			imageName,
		).RunSuccess()
		if err != nil {
			return err
		}
	}
	return nil
}
