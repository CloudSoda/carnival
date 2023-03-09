package samba

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"cloudsoda.dev/carnival/pkg/ansi"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

func isMedia(name string) bool {
	suffixes := []string{".mp4", ".mkv", ".webm", ".webp", ".jpg", ".jpeg", ".gif", ".png", ".mp3", ".aac", "mov", ".avi", ".wmv", ".mpeg", ".vob"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func List(cliCtx *cli.Context) error {
	u, err := urlFromContext(cliCtx)
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

	infos, err := share.ReadDir(u.Path)
	if err != nil {
		return fmt.Errorf("listing '%s': %v", u.Path, err)
	}
	slices.SortFunc(infos, func(a, b fs.FileInfo) bool {
		return a.Name() < b.Name()
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', tabwriter.AlignRight)
	currYear := time.Now().In(time.Local).Year()
	for _, info := range infos {
		mtime := info.ModTime().In(time.Local)
		monthDay := mtime.Format("Jan\t02")
		year := mtime.Year()
		var date2 string
		if year == currYear {
			date2 = fmt.Sprintf("%02d:%02d", mtime.Hour(), mtime.Minute())
		} else {
			date2 = strconv.Itoa(mtime.Year())
		}
		var name string
		if info.Mode().IsDir() {
			name = ansi.Blue(info.Name())
		} else if isMedia(info.Name()) {
			name = ansi.Purple(info.Name())
		} else if info.Mode()&111 != 0 {
			name = ansi.Cyan(info.Name())
		} else {
			name = info.Name()
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t %s\n",
			info.Mode().String(),
			strconv.FormatInt(info.Size(), 10),
			monthDay,
			date2,
			name)
	}
	tw.Flush()

	return nil
}
