package samba

import (
	"context"

	"github.com/cloudsoda/go-smb2"
	"github.com/urfave/cli/v2"
)

func parseOptions(ctx *cli.Context) []smb2.MountOption {
	var mos []smb2.MountOption
	if ctx.Bool(FlagMapchars) {
		mos = append(mos, smb2.WithMapChars())
	}
	if ctx.Bool(FlagMapposix) {
		mos = append(mos, smb2.WithMapPosix())
	}
	return mos
}

func connect(u URL, domain string) (*smb2.Session, error) {
	d := &smb2.Dialer{}
	if u.Credentials != nil {
		d.Initiator = &smb2.NTLMInitiator{
			User:     u.Credentials.Username,
			Password: u.Credentials.Password,
			Domain:   domain,
		}
	} else {
		d.Initiator = &smb2.NTLMInitiator{}
	}

	srvr, err := d.Dial(context.Background(), u.Address)
	if err != nil {
		return nil, err
	}

	return srvr, nil
}
