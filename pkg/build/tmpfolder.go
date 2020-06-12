/*
 Copyright 2020 Qiniu Cloud (qiniu.com)

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package build

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiniu/goc/pkg/cover"
)

func (b *Build) MvProjectsToTmp() (newGopath string, newWorkingDir string, tmpBuildDir string, pkgs map[string]*cover.Package) {
	listArgs := []string{"-json"}
	if len(b.BuildFlags) != 0 {
		listArgs = append(listArgs, b.BuildFlags)
	}
	listArgs = append(listArgs, "./...")
	b.Pkgs = cover.ListPackages(".", strings.Join(listArgs, " "), "")

	b.mvProjectsToTmp()
	oriGopath := os.Getenv("GOPATH")
	if b.IsMod == true {
		b.NewGOPATH = ""
	} else if oriGopath == "" {
		b.NewGOPATH = b.TmpDir
	} else {
		b.NewGOPATH = fmt.Sprintf("%v:%v", b.TmpDir, oriGopath)
	}
	log.Printf("New GOPATH: %v", b.NewGOPATH)
	return
}

func (b *Build) mvProjectsToTmp() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatalf("Cannot get current working directoy, the error is: %v", err)
	}
	b.TmpDir = filepath.Join(os.TempDir(), TmpFolderName(path))

	// Delete previous tmp folder and its content
	os.RemoveAll(b.TmpDir)
	// Create a new tmp folder
	err = os.MkdirAll(filepath.Join(b.TmpDir, "src"), os.ModePerm)
	if err != nil {
		log.Fatalf("Fail to create the temporary build directory. The err is: %v", err)
	}
	log.Printf("Tmp project generated in: %v", b.TmpDir)

	// set Build.IsMod flag, so we dont have to call checkIfLegacyProject another time
	if b.checkIfLegacyProject() {
		b.cpLegacyProject()
	} else {
		b.IsMod = true
		b.cpGoModulesProject()
	}
	b.getTmpwd()

	log.Printf("New workingdir in tmp directory in: %v", b.TmpWorkingDir)
}

func TmpFolderName(path string) string {
	sum := sha256.Sum256([]byte(path))
	h := fmt.Sprintf("%x", sum[:6])

	return "goc-" + h
}

// checkIfLegacyProject Check if it is go module project
// true legacy
// false go mod
func (b *Build) checkIfLegacyProject() bool {
	for _, v := range b.Pkgs {
		if v.Module == nil {
			return true
		}
		return false
	}
	log.Fatalln("Should never be reached....")
	return false
}

// getTmpwd get the corresponding working directory in the temporary working directory
// and store it in the Build.tmpWorkdingDir
func (b *Build) getTmpwd() {
	for _, pkg := range b.Pkgs {
		path, err := os.Getwd()
		if err != nil {
			log.Fatalf("Cannot get current working directory, the error is: %v", err)
		}

		index := -1
		var parentPath string
		if b.IsMod == false {
			index = strings.Index(path, pkg.Root)
			parentPath = pkg.Root
		} else {
			index = strings.Index(path, pkg.Module.Dir)
			parentPath = pkg.Module.Dir
		}

		if index == -1 {
			log.Fatalf("goc install not executed in project directory.")
		}
		b.TmpWorkingDir = filepath.Join(b.TmpDir, path[len(parentPath):])
		// log.Printf("New building directory in: %v", tmpwd)
		return
	}

	log.Fatalln("Should never be reached....")
	return
}

func (b *Build) findWhereToInstall() string {
	if GOBIN := os.Getenv("GOBIN"); GOBIN != "" {
		return GOBIN
	}

	// old GOPATH dir
	GOPATH := os.Getenv("GOPATH")
	if false == b.IsMod {
		for _, v := range b.Pkgs {
			return filepath.Join(v.Root, "bin")
		}
	}
	if GOPATH != "" {
		return filepath.Join(strings.Split(GOPATH, ":")[0], "bin")
	}
	return filepath.Join(os.Getenv("HOME"), "go", "bin")
}
