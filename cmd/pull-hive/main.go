// pull-hive copies the pieces sorters reported to a Hive into the data
// folder's hive.sqlite: every machine the signed-in user may read. It only
// reads from Hive. Restart pile to see what it pulled.
//
// The sign-in comes from HIVE_EMAIL and HIVE_PASSWORD, or with -login-stdin
// from two lines on stdin (the email, then the password), so a password never
// rides on a command line.
package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/basicallysource/pile/internal/hive"
)

func main() {
	data := flag.String("data", "data", "the data folder; the pull goes to hive.sqlite in it")
	base := flag.String("hive", "https://hive.basically.website", "the Hive to pull from")
	full := flag.Bool("full", false, "pull every piece again, not only the new ones")
	stdin := flag.Bool("login-stdin", false, "read the email and the password from stdin, one a line")
	flag.Parse()

	email, password := os.Getenv("HIVE_EMAIL"), os.Getenv("HIVE_PASSWORD")
	if *stdin {
		sc := bufio.NewScanner(os.Stdin)
		var lines []string
		for len(lines) < 2 && sc.Scan() {
			lines = append(lines, strings.TrimRight(sc.Text(), "\r"))
		}
		if len(lines) == 2 {
			email, password = lines[0], lines[1]
		}
	}
	if email == "" || password == "" {
		log.Fatal("no sign-in: set HIVE_EMAIL and HIVE_PASSWORD, or pass -login-stdin")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	c, err := hive.SignIn(ctx, *base, email, password)
	if err != nil {
		log.Fatal(err)
	}
	s, err := hive.Open(filepath.Join(*data, "hive.sqlite"))
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	total := 0
	err = hive.Pull(ctx, c, s, *full, func(name string, n int) {
		total += n
		log.Printf("%s: %d pieces read", name, n)
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("pulled %d pieces", total)
}
