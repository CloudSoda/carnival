package samba

import (
	"errors"
	"fmt"
	"os"

	"github.com/cloudsoda/go-smb2"
	"github.com/urfave/cli/v2"
)

func Sd(ctx *cli.Context) error {
	u, err := urlFromContext(ctx)
	if err != nil {
		return err
	}
	if u.Share == "" {
		return errors.New("no share name specified")
	}

	session, err := connect(u, ctx.String(FlagDomain))
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}
	defer session.Logoff()

	share, err := session.Mount(u.Share, parseOptions(ctx)...)
	if err != nil {
		return fmt.Errorf("mounting '%s': %v", u.Share, err)
	}
	defer share.Umount()

	// try to read all security descriptors
	sd, err := share.SecurityInfo(u.Path, smb2.OwnerSecurityInformation|smb2.GroupSecurityInformation|smb2.DACLSecurityInformation|smb2.SACLSecurityInformation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "W: failed to read full security descriptor: %v\n", err)
		fmt.Fprintf(os.Stderr, "W: trying to read all but SACL\n")
		// try to read all but SACL
		sd, err = share.SecurityInfo(u.Path, smb2.OwnerSecurityInformation|smb2.GroupSecurityInformation|smb2.DACLSecurityInformation)
		if err != nil {
			return fmt.Errorf("read security descriptors: %v", err)
		}
	}

	var sdStr string
	if ctx.Bool("pretty") {
		sdStr = sd.StringIndent(0)
	} else {
		sdStr = sd.String()
	}

	fmt.Println(sdStr)

	return nil
}
