package samba

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"
)

func destinationPath(inPath, dst string) (string, error) {
	if dst == "" {
		return "", errors.New("destination path required")
	}

	if dst == "." {
		// use the source file name
		srcPath := path.Base(inPath)
		if srcPath == "" || srcPath == "." || srcPath == "/" {
			return "", errors.New("invalid source path")
		}
		dst = srcPath
	}

	return dst, nil
}

func Copy(ctx *cli.Context) error {
	u, err := urlFromContext(ctx)
	if err != nil {
		return err
	}
	if u.Share == "" {
		return errors.New("no share name specified")
	}

	dstPath, err := destinationPath(u.Path, ctx.Args().Get(1))
	if err != nil {
		return err
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

	info, err := share.Stat(u.Path)
	if err != nil {
		return fmt.Errorf("stat: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("source is a directory")
	}

	src, err := share.Open(u.Path)
	if err != nil {
		return fmt.Errorf("open source file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("creating destination file: %v", err)
	}
	defer dst.Close()

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	sp.Spinner = spinner.Dot

	stop := make(chan bool)
	go func() {
		fmt.Printf("%s", sp.View())
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sp, _ = sp.Update(sp.Tick())
				fmt.Printf("\r%s", sp.View())
			case <-stop:
				return
			}
		}
	}()

	start := time.Now()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy error: %v", err)
	}
	dur := time.Since(start)

	if err := dst.Close(); err != nil {
		return fmt.Errorf("closing destination file: %v", err)
	}
	close(stop)

	speedInBytes := float64(info.Size()) / dur.Seconds()
	speedInMBytes := speedInBytes / 1024 / 1024
	fmt.Fprintf(os.Stdout, "\rTook %v. %v MB/s\n", dur, speedInMBytes)

	return nil
}
