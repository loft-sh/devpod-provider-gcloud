package main

import (
	"math/rand"
	"time"

	"github.com/loft-sh/devpod-gcloud-provider/cmd"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	cmd.Execute()
}
