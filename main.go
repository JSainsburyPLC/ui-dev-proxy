package main

import (
	"github.com/JSainsburyPLC/ui-dev-proxy/commands"
	"github.com/JSainsburyPLC/ui-dev-proxy/file"
	"github.com/urfave/cli"
	"log"
	"os"
)

const version = "0.1.0"

func main() {
	app := cli.NewApp()
	app.Name = "ui-dev-proxy"
	app.Version = version

	logger := log.New(os.Stdout, "", log.LstdFlags)
	app.Writer = logger.Writer()

	confProvider := file.ConfigProvider()

	app.Commands = []cli.Command{
		commands.StartCommand(logger, confProvider),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
