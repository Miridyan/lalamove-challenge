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
	var versionSlice []*semver.Version
	var lastCommmitted *semver.Version

	// Instead of creating a new slice identical to the input and sorting it, I could just sort the input array to save time, however
	// I feel that this is the wrong choice because that would modify data outside of the function. Although this program is simple
	// and this isn't really an issue, I feel that it is best that all mutations that occur within a function should be isolated to it
	// in order to mitigate side effects.
	sortedReleases := make([]*semver.Version, len(releases))
	copy(sortedReleases, releases)
	semver.Sort(sortedReleases)

	// Iterate along the slice backwards and add some element to `versionSlice` if the element that comes after it has a greater
	// major or minor version, but only if that element is greater than `minVersion`.
	for i := len(sortedReleases) - 1; i >= 0; i-- {
		// The following if statements could be reduced in number, but I believe that this
		// formatting is more legible and makes the conditions more easily understood.
		if !sortedReleases[i].LessThan(*minVersion) && len(sortedReleases[i].PreRelease) == 0 {
			// If `versionSlice` has nothing in it then it whatever is currently selected must be the greatest
			// version of some release. If the currently selected element has a lesser major or minor version
			// than `lastCommitted` then the currently selected element must be the latest release of a different
			// version. At each of these points update `lastCommited`.
			if len(versionSlice) == 0 {
				versionSlice = append(versionSlice, sortedReleases[i])
				lastCommmitted = sortedReleases[i]
			} else if lastCommmitted.Major > sortedReleases[i].Major || lastCommmitted.Minor > sortedReleases[i].Minor {
				versionSlice = append(versionSlice, sortedReleases[i])
				lastCommmitted = sortedReleases[i]
			}
		}
	}
	return versionSlice
}

// ParseFile takes in the path to the input file and returns an array of Repo structs
func ParseFile(path string) []Repo {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("[File Read Error] %s\n", err)
		return nil
	}

	// The file format should have all elements in the format `user/repository,minVersion\n` so we split at `/`, `,`, and `\n` in order to
	// convert it to an intermediate slice of {"user", "repository", "minVersion", ...}
	intermediate := strings.FieldsFunc(string(data), func(r rune) bool {
		return r == '\n' || r == ',' || r == '/'
	})
	intermediate = intermediate[2:]

	// Since the intermediate slice has 3 elements that should correspond to a single `Repo` struct, the size of the `fileSlice` slice
	// will be 1/3rd the size of the intermediate slice. Populate each `Repo` struct with the entries from intermediate array.
	// In theory you could just return the intermediate string and do this work later on, but the current approach could simplify
	// logic later on and allow for error checking in the parsing stage.
	fileSlice := make([]Repo, len(intermediate)/3)
	for i := 0; i < len(intermediate); i += 3 {
		// If the conversion to a repo struct fails then it will throw a panic for either index out of bounds or for creating a new
		// semver.Version struct. In either case, catch the panic and inform the user which line in the file has improper formatting.
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("[File Conversion Error] Improper file formatting at line %d\n", i/3+1)
				return
			}
		}()

		fileSlice[i/3] = Repo{
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
		return
	}

	// Github
	client := github.NewClient(nil)
	ctx := context.Background()
	opt := &github.ListOptions{PerPage: 500}

	for _, repo := range repos {
		releases, _, err := client.Repositories.ListReleases(ctx, repo.Path[0], repo.Path[1], opt)

		if err != nil {
			// panic(err) // is this really a good way?
			fmt.Println("[Github Repository Error]", err)
			return
		}

		minVersion := repo.MinVersion
		allReleases := make([]*semver.Version, len(releases))

		for i, release := range releases {
			versionString := *release.TagName
			if versionString[0] == 'v' {
				versionString = versionString[1:]
			}

			// It seems that this is necessary because it is not a guarentee that a github repository has their releases in a
			// format supported by `coreos/go-semver`. Also, if the releases themselves are formatted incorrectly on the gihub
			// site, the `releases` slice will be empty.
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("[Semver Error] %s is not a supported go-semver format!\n", versionString)
				}
			}()

			allReleases[i] = semver.New(versionString)
		}

		versionSlice := LatestVersions(allReleases, minVersion)

		fmt.Printf("latest versions of %s/%s: %s\n", repo.Path[0], repo.Path[1], versionSlice)
	}
}
