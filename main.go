package main

import (
	"fmt"
	"os"
	"runtime"

	"cloudsoda.dev/carnival/pkg/samba"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02T15:04:05",
	}).With().Timestamp().Logger()

	prog := cli.NewApp()
	prog.Name = "Carnival"
	prog.HelpName = "carnival"
	prog.Usage = "For testing connectivity to SMB shares"

	prog.Commands = []*cli.Command{
		{
			Name:      "cp",
			Usage:     "copy a file from a samba share to a local destination",
			UsageText: "carnival cp [smburl] [destination]",
			Action:    samba.Copy,
		},
		{
			Name:      "help",
			Usage:     "show the app or a command's help text",
			UsageText: "carnival help [command]",
			Action:    prog.Action, // the default action is help
		},
		{
			Name:      "ls",
			Usage:     "list the contents of a directory",
			UsageText: "carnival ls [smburl]",
			Action:    samba.List,
		},
		{
			Name:      "md5",
			Usage:     "calculate the md5 hash of a file",
			UsageText: "carnival md5 [smburl]",
			Action:    samba.MD5,
		},
		{
			Name:      "shares",
			Usage:     "list the publicly visible shares",
			UsageText: "carnival shares [smburl]",
			Action:    samba.Shares,
		},
	}

	if err := prog.Run(os.Args); err != nil {
		if runtime.GOOS != "windows" {
			fmt.Fprintln(os.Stderr, err)
		} else {
			// cmd & powershell automatically write a newline
			// after the program terminates
			fmt.Fprint(os.Stderr, err)
		}
		os.Exit(1)
	}
}
