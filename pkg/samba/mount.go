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

func connect(ctx *cli.Context, u URL) (*smb2.Session, error) {
	d := &smb2.Dialer{}

	if ctx.Bool(FlagKerberos) {
		initiator, err := newKrb5Initiator(ctx, u)
		if err != nil {
			return nil, err
		}
		d.Initiator = initiator
	} else if u.Credentials != nil {
		d.Initiator = &smb2.NTLMInitiator{
			User:     u.Credentials.Username,
			Password: u.Credentials.Password,
			Domain:   ctx.String(FlagDomain),
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
