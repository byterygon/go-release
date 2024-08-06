package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"sync"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

var osList = []string{"linux"}
var archList = []string{"amd64", "arm64"}
var kindList = []string{"archive"}
var minimunVersion = "go1.19.13"

type GoVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []struct {
		Filename string `json:"filename"`
		Os       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		Sha256   string `json:"sha256"`
		Size     int    `json:"size"`
		Kind     string `json:"kind"`
	} `json:"files"`
}

func checkIfError(err error) {
	if err != nil {

		log.Fatal(err)
	}
}
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func main() {
	tags := []string{}

	var wg sync.WaitGroup
	var goVersions = []GoVersion{}
	r, err := git.PlainOpen(".")

	wg.Add(1)
	go func() {

		checkIfError(err)

		tagrefs, err := r.Tags()
		checkIfError(err)
		err = tagrefs.ForEach(func(t *plumbing.Reference) error {
			tag := ""
			fmt.Sscanf(string(t.Name()), "refs/tags/v%s", &tag)
			tags = append(tags, tag)
			return nil
		})
		checkIfError(err)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		resp, err := http.Get("https://go.dev/dl/?mode=json&include=all")
		checkIfError(err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		checkIfError(err)

		json.Unmarshal(body, &goVersions)
		wg.Done()
	}()
	wg.Wait()

	for i := 0; i < len(goVersions); i++ {
		goVersion := goVersions[i]
		if minimunVersion == goVersion.Version {
			break
		}
		if !goVersion.Stable {
			continue
		} else {
			goTag := ""
			fmt.Sscanf(goVersion.Version, "go%s", &goTag)
			if !slices.Contains(tags, goTag) {
				downloadedFile := make([]string, 0)
				for j := 0; j < len(goVersions[i].Files); j++ {
					file := goVersions[i].Files[j]
					if slices.Contains(osList, file.Os) && slices.Contains(archList, file.Arch) && slices.Contains(kindList, file.Kind) {
						filePath := fmt.Sprintf("./%s", goVersions[i].Files[j].Filename)
						if _, err := os.Stat(filePath); err == nil {
							// path/to/whatever exists

						} else if errors.Is(err, os.ErrNotExist) {
							// path/to/whatever does *not* exist

							downloadFile(filePath, fmt.Sprintf("https://go.dev/dl/%s", goVersions[i].Files[j].Filename))
							downloadedFile = append(downloadedFile, goVersions[i].Files[j].Filename)
						} else {
							// Schrodinger: file may or may not exist. See err for details.

							// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
							log.Fatal(err)

						}
					}
				}

				cmd := exec.Command("git", "config", "--global", "user.email", "tranuyson@gmail.com")
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				cmd = exec.Command("git", "config", "--global", "user.name", "byterygon")
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				cmd = exec.Command("git", "add", ".")
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Release %s", goVersion.Version))
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				checkIfError(err)
				cmd = exec.Command("git", "push", "-u", "origin", "HEAD")
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				cmd = exec.Command("git", "tag", fmt.Sprintf("v%s", goTag))
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				cmd = exec.Command("git", "push", "--tag")
				if err := cmd.Run(); err != nil {
					log.Fatal(err)
				}
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				for f := 0; f < len(downloadedFile); f++ {
					cwd, _ := os.Getwd()
					err := os.Remove(fmt.Sprintf("%s/%s", cwd, downloadedFile[f]))
					checkIfError(err)
				}
			}
		}
	}

}

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
