package main

import (
	"encoding/json"
	"os"

	"github.com/orange-cloudfoundry/gitlab-release-resource"
)

func main() {
	request := resource.NewCheckRequest()
	inputRequest(&request)

	gitlab, err := resource.NewGitLabClient(request.Source)
	if err != nil {
		resource.Fatal("constructing gitlab client", err)
	}

	command := resource.NewCheckCommand(gitlab)
	response, err := command.Run(request)
	if err != nil {
		resource.Fatal("running command", err)
	}

	outputResponse(response)
}

func inputRequest(request *resource.CheckRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		resource.Fatal("reading request from stdin", err)
	}
}

func outputResponse(response []resource.Version) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		resource.Fatal("writing response to stdout", err)
	}
}
