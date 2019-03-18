package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/google/go-github/github"
)

// Repo is a struct to represent each repository parsed from the input file. The path will have the format `user/repo` to append to the end of the
// github URL and the min version is just a semver Version struct
type Repo struct {
	Path       [2]string
	MinVersion *semver.Version
}

// LatestVersions returns a sorted slice with the highest version as its first element and the highest version of the smaller minor versions in a descending order
func LatestVersions(releases []*semver.Version, minVersion *semver.Version) []*semver.Version {
	sortedReleases := make([]*semver.Version, len(releases))
	copy(sortedReleases, releases)
	semver.Sort(sortedReleases)

	versionSlice := []*semver.Version{sortedReleases[len(sortedReleases)-1]}
	curMaj := sortedReleases[len(sortedReleases)-1].Major
	curMin := sortedReleases[len(sortedReleases)-1].Minor

	for i := len(sortedReleases) - 1; i >= 0; i-- {
		if i > 0 {
			if sortedReleases[i-1].Major < curMaj || sortedReleases[i-1].Minor < curMin {
				if minVersion.LessThan(*sortedReleases[i-1]) {
					versionSlice = append(versionSlice, sortedReleases[i-1])
				}
				curMaj = sortedReleases[i-1].Major
				curMin = sortedReleases[i-1].Minor
			}
		}
	}
	return versionSlice
}

// ParseFile takes in the path to the input file and returns an array of Repo structs
func ParseFile(path string) []*Repo {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("[File Read Error]", err)
		return nil
	}

	intermediate := strings.FieldsFunc(string(data), func(r rune) bool {
		return r == '\n' || r == ',' || r == '/'
	})
	fileSlice := make([]*Repo, len(intermediate)/3)

	for i := 0; i < len(intermediate); i += 3 {
		fileSlice[i/3] = &Repo{
			Path:       [2]string{intermediate[i], intermediate[i+1]},
			MinVersion: semver.New(intermediate[i+2]),
		}
	}

	return fileSlice
}

// Here we implement the basics of communicating with github through the library as well as printing the version
// You will need to implement LatestVersions function as well as make this application support the file format outlined in the README
// Please use the format defined by the fmt.Printf line at the bottom, as we will define a passing coding challenge as one that outputs
// the correct information, including this line
func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Please provide path to the library releases file")
		return
	}

	repos := ParseFile(args[0])
	if repos == nil {
		fmt.Println("No libraries provided in file")
		return
	}

	fmt.Println(repos)

	// Github
	client := github.NewClient(nil)
	ctx := context.Background()
	opt := &github.ListOptions{PerPage: 500}
	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", opt)

	if err != nil {
		// panic(err) // is this really a good way?
		fmt.Println("[Github Repository Error]", err)
		return
	}

	minVersion := semver.New("1.8.0")
	allReleases := make([]*semver.Version, len(releases))

	for i, release := range releases {
		versionString := *release.TagName

		if versionString[0] == 'v' {
			versionString = versionString[1:]
		}
		allReleases[i] = semver.New(versionString)
	}
	versionSlice := LatestVersions(allReleases, minVersion)

	fmt.Printf("latest versions of kubernetes/kubernetes: %s\n", versionSlice)
}
