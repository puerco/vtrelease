package main

import (
	"github.com/puerco/vtrelease/pkg/commands"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := commands.New().Execute(); err != nil {
		logrus.Fatalf("error during command execution: %v", err)
	}
}
