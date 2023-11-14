package samba

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func Shares(ctx *cli.Context) error {
	u, err := urlFromContext(ctx)
	if err != nil {
		return err
	}
	session, err := connect(u, ctx.String(FlagDomain))
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}
	defer session.Logoff()

	names, err := session.ListSharenames()
	if err != nil {
		return err
	}

	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}
