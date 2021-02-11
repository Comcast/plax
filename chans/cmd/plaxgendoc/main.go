package test

import (
	"github.com/Comcast/plax/chans"
	_ "github.com/Comcast/plax/chans/sqs"
)

func main() {
	chans.GenDocs("..")
}
