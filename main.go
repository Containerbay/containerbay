package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mholt/archiver/v3"
	"github.com/mudler/containerbay/api"
	"github.com/mudler/containerbay/internal"
	terminal "github.com/mudler/go-isterminal"
	"github.com/mudler/luet/pkg/api/core/image"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/urfave/cli"
)

var flags = []cli.Flag{
	&cli.BoolFlag{
		Name:   "gzip",
		Usage:  "enable gzip",
		EnvVar: "CONTAINERBAY_GZIP",
	},
	&cli.BoolFlag{
		Name:   "debug",
		Usage:  "enable debug messages",
		EnvVar: "DEBUG",
	},
	&cli.StringFlag{
		Name:   "address",
		Usage:  "listening address",
		EnvVar: "CONTAINERBAY_LISTENADDR",
		Value:  ":8080",
	},
	&cli.StringFlag{
		Name:   "dns",
		Usage:  "magic dns address",
		EnvVar: "CONTAINERBAY_MAGICDNS",
	},
	&cli.StringFlag{
		Name:   "default-image",
		Usage:  "Default image to use",
		EnvVar: "CONTAINERBAY_DEFAULTIMAGE",
		Value:  "ghcr.io/containerbay/containerbay.io:latest",
	},
	&cli.StringFlag{
		Name:   "cleanup",
		Usage:  "store cleanup interval",
		EnvVar: "CONTAINERBAY_CLEANUPINTERVAL",
	},
	&cli.StringFlag{
		Name:   "store",
		Usage:  "cachestore directory",
		EnvVar: "CONTAINERBAY_CACHEDIR",
		Value:  "/tmp/containerbay",
	},
	&cli.StringFlag{
		Name:   "max-size",
		Usage:  "Max imagesize to serve",
		EnvVar: "CONTAINERBAY_MAXSIZE",
	},
	&cli.IntFlag{
		Name:   "workers",
		Usage:  "download workers",
		Value:  10,
		EnvVar: "CONTAINERBAY_WORKERS",
	},
	&cli.IntFlag{
		Name:   "pool",
		Usage:  "Workers pool size",
		Value:  0,
		EnvVar: "CONTAINERBAY_POOLSIZE",
	},
}

type ptermWriter struct {
	Printer pterm.PrefixPrinter
}

func (p *ptermWriter) Write(b []byte) (int, error) {
	p.Printer.Println(string(b))
	return len(b), nil
}

func echoConfig(c *cli.Context) func(e *echo.Echo) error {
	return func(e *echo.Echo) error {
		if c.Bool("gzip") {
			e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
				Level: 5,
			}))
		}
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: &ptermWriter{Printer: pterm.Info}}))
		return nil
	}
}

func startBanner() {
	pterm.Info.Println("Starting Containerbay")
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	if !terminal.IsTerminal(os.Stdout) {
		pterm.DisableColor()
		pterm.DisableStyling()
	}

	app := &cli.App{
		Name:    "containerbay",
		Author:  "Ettore Di Giacinto",
		Usage:   "containerbay",
		Version: fmt.Sprintf("%s-g%s", internal.Version, internal.Commit),
		Commands: []cli.Command{
			{
				Name:    "pack",
				Aliases: []string{"p"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "destination",
						Usage: "Destination image tar",
						Value: "output.tar",
					},
					&cli.StringFlag{
						Name:  "os",
						Value: runtime.GOOS,
						Usage: "Overrides default image OS",
					},
					&cli.StringFlag{
						Name:  "arch",
						Value: runtime.GOARCH,
						Usage: "Overrides default image ARCH",
					},
				},
				UsageText: `
Packs files inside a tar which is consumable by docker.

E.g.
$ containerbay --destination out.tar foo/image:tar srcfile1 srcfile2 srcdir1 ...
$ docker load -i out.tar
$ docker push foo/image:tar ...
`,
				Usage: "pack a directory as a container image",
				Action: func(c *cli.Context) error {
					if !c.Args().Present() {
						return errors.New("need an image and source files to include inside the tar")
					}
					if c.Bool("debug") {
						pterm.EnableDebugMessages()
					}
					dst := c.String("destination")
					img := c.Args().First()
					src := c.Args().Tail()

					dir, err := os.MkdirTemp("", "containerbay")
					if err != nil {
						return err
					}
					defer os.RemoveAll(dir)

					err = archiver.Archive(src, filepath.Join(dir, "archive.tar"))
					if err != nil {
						return err
					}
					pterm.Info.Printfln("Creating '%s' as '%s' from %v", dst, img, src)
					return image.CreateTar(filepath.Join(dir, "archive.tar"), dst, img, c.String("arch"), c.String("os"))
				},
			},
			{
				Name:    "standalone",
				Aliases: []string{"s"},
				Flags:   flags,
				UsageText: `
Runs an image over API

E.g.
$ containerbay standalone foo/image:tag
`,
				Usage: "run the daemon to serve only a single container image",
				Action: func(c *cli.Context) error {
					if !c.Args().Present() {
						return errors.New("need an image")
					}
					if c.Bool("debug") {
						pterm.EnableDebugMessages()
					}
					startBanner()
					return api.New(
						api.WithListeningAddress(c.String("address")),
						api.WithMagicDNS(c.String("dns")),
						api.WithCacheStore(c.String("store")),
						api.Standalone(c.Args().First()),
						api.WithMaxSize(c.String("max-size")),
						api.WithWorkers(c.Int("workers")),
						api.WithCleanupInterval(c.String("cleanup")),
						api.WithPoolSize(c.Int("pool")),
						api.WithDefaultImage(c.String("default-image")),
					).Start(echoConfig(c))
				},
			},
			{
				Name:    "run",
				Aliases: []string{"r"},
				Flags:   flags,
				Usage:   "run the api to serve multiple container images",
				Action: func(c *cli.Context) error {
					if c.Bool("debug") {
						pterm.EnableDebugMessages()
					}
					startBanner()
					return api.New(
						api.WithListeningAddress(c.String("address")),
						api.WithCacheStore(c.String("store")),
						api.WithMagicDNS(c.String("dns")),
						api.WithMaxSize(c.String("max-size")),
						api.WithWorkers(c.Int("workers")),
						api.WithCleanupInterval(c.String("cleanup")),
						api.WithPoolSize(c.Int("pool")),
						api.WithDefaultImage(c.String("default-image")),
					).Start(echoConfig(c))
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		pterm.Fatal.Println(err)
	}
}
