package samba

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/urfave/cli/v2"
)

func MD5(ctx *cli.Context) error {
	u, err := urlFromContext(ctx)
	if err != nil {
		return err
	}
	if u.Share == "" {
		return errors.New("no share name specified")
	}

	session, err := connect(u)
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}
	defer session.Logoff()

	share, err := session.Mount(u.Share)
	if err != nil {
		return fmt.Errorf("mounting '%s': %v", u.Share, err)
	}

	f, err := share.Open(u.Path)
	if err != nil {
		return fmt.Errorf("open file failed: %v", err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("read error: %v", err)
	}

	fmt.Println(hex.EncodeToString(h.Sum(nil)), "  ", u.Path)

	return nil
}
