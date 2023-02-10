package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	containerdarchive "github.com/containerd/containerd/archive"
	"github.com/docker/go-units"
	"github.com/lthibault/jitterbug"
	"github.com/moby/moby/api/types"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/labstack/echo/v4"
	"github.com/mudler/containerbay/store"
)

// API returns a new containerbay instance
type API struct {
	listenAddr        string
	dnsTXTKey         string
	magicDNS          string
	standaloneImage   string
	defaultImage      string
	cacheStore        *store.Store
	maxSize           int64
	poolSize, workers int
	pool              chan workPackage
	cleanupInterval   time.Duration
	auth              *types.AuthConfig
}

type workPackage struct {
	img, dst string
}

func (a *API) downloadImage(image, dst string) error {
	os.MkdirAll(dst, os.ModePerm)
	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	var img v1.Image

	if a.auth != nil {
		img, err = remote.Image(ref, remote.WithAuth(staticAuth{a.auth}))
		if err != nil {
			return err
		}
	} else {
		img, err = remote.Image(ref)
		if err != nil {
			return err
		}
	}

	reader := mutate.Extract(img)

	defer reader.Close()

	_, err = containerdarchive.Apply(context.Background(), dst, reader)
	if err != nil {
		return err
	}
	return nil
}

func imageSize(img v1.Image) (size int64) {
	lyrs, _ := img.Layers()
	for _, l := range lyrs {
		s, _ := l.Size()
		size += s
	}
	return
}

type errorMessage struct {
	Error string `json:"error"`
}

func retError(c echo.Context, template string, i ...interface{}) error {
	return c.JSON(http.StatusInternalServerError, errorMessage{Error: fmt.Sprintf(template, i...)})
}

func (a *API) startWorkers() {
	pterm.Info.Printfln("Starting '%d' workers", a.workers)

	for i := 0; i < a.workers; i++ {
		go func() {
			for f := range a.pool {
				a.downloadImage(f.img, f.dst)
			}
		}()
	}
}

func (a *API) cleanupWorker(ctx context.Context) {
	if a.cleanupInterval == 0 {
		pterm.Info.Println("Cleanup disabled")
		return
	}
	pterm.Info.Println("Cleanups enabled")

	go func() {
		t := jitterbug.New(
			a.cleanupInterval,
			&jitterbug.Norm{Stdev: time.Second * 10},
		)
		for {
			select {
			case <-t.C:
				err := a.cacheStore.Clean()
				if err != nil {
					pterm.Error.Println("error while cleaning up:", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (a *API) renderImage(c echo.Context, image, strip string) error {

	ref, err := name.ParseReference(image)
	if err != nil {
		return retError(c, "while parsing image reference '%s'", err.Error())
	}

	img, err := remote.Image(ref)
	if err != nil {
		return retError(c, "while fetching remote image reference '%s'", err.Error())
	}

	size := imageSize(img)
	if a.maxSize != 0 && size > a.maxSize {
		pterm.Warning.Printfln("Refusing to serve image '%s' (size: %s)\n", image, units.HumanSize(float64(size)))
		return retError(c, "max size exceeded: image %d, threshold %d", size, a.maxSize)
	}

	pterm.Info.Printfln("Serving image: %s Size: %s\n", image, units.HumanSize(float64(size)))

	h, err := img.Digest()
	if err != nil {
		return retError(c, "while getting image digest: %s", err.Error())
	}

	// If doesn't exist in cache we have to download it
	// We let the worker download them, and handle the request separately
	if !a.cacheStore.Exists(h.Hex) {
		pterm.Info.Printfln("Not present in cache %s: %s Size: %s\n", h.Hex, image, units.HumanSize(float64(size)))
		a.pool <- workPackage{img: image, dst: a.cacheStore.Path(h.Hex)}
		return c.HTML(202, "Processing")
	}

	return echo.WrapHandler(
		http.StripPrefix(strip, http.FileServer(http.Dir(a.cacheStore.Path(h.Hex)))))(c)
}

func (a *API) containerFromDomain(domain string) (string, error) {

	pterm.Debug.Printfln("(magicDNS) Querying TXT records for domain '%s'", domain)

	txtrecords, err := net.LookupTXT(domain)
	if err != nil {
		return "", err
	}

	pterm.Debug.Printfln("(magicDNS) Found '%v' for domain '%s'", txtrecords, domain)

	for _, txt := range txtrecords {
		dat := strings.Split(txt, " ")
		pterm.Debug.Printfln("(magicDNS) txt record '%v' for domain '%s'", txt, domain)

		if len(dat) > 0 {
			record := dat[0]
			//	dd := dat[1]
			// if dd != domain { continue }
			r := strings.Split(record, "=")
			if len(r) > 0 {
				if r[0] == a.dnsTXTKey {
					return r[1], nil
				}
			}
		}
	}
	return "", errors.New("record not found")
}

// EchoOption is a generic handler which mutates the underlying Echo instance
type EchoOption func(e *echo.Echo) error

// Start starts the API with the given EchoOption
func (a *API) Start(opts ...EchoOption) error {

	a.pool = make(chan workPackage, a.poolSize)
	a.startWorkers()
	a.cleanupWorker(context.Background())
	a.cacheStore.CleanAll()

	ec := echo.New()
	for _, o := range opts {
		o(ec)
	}

	pterm.Info.Printfln("Cachestore dir at '%s'", a.cacheStore)
	pterm.Info.Printfln("Max image size '%s'", units.HumanSize(float64(a.maxSize)))
	pterm.Info.Printfln("Default image '%s'", a.defaultImage)

	if a.standaloneImage != "" {
		ec.GET("/*", func(c echo.Context) error {
			return a.renderImage(c, a.standaloneImage, "/")
		})
	} else {
		ec.GET("/*", func(c echo.Context) error {
			req := c.Request()
			host := req.Host
			if a.magicDNS != "" {
				dns := a.magicDNS
				if !strings.HasPrefix(dns, ".") {
					dns = "." + dns
				}
				hostdots := strings.ReplaceAll(host, dns, "")
				fields := strings.Split(hostdots, ".")

				pterm.Info.Printfln("Trying to resolve magicDNS(%s) for '%s'", dns, host)

				// A host can be: registry.org.container.tag
				if len(fields) >= 4 {
					var tag, org, container, registry string

					// pop the required fields
					tag, fields = fields[len(fields)-1], fields[:len(fields)-1]
					container, fields = fields[len(fields)-1], fields[:len(fields)-1]
					org, fields = fields[len(fields)-1], fields[:len(fields)-1]

					registry = strings.Join(fields, ".") // recompose registry

					// compose image name
					image := fmt.Sprintf("%s/%s/%s:%s", registry, org, container, tag)
					pterm.Info.Printfln("magicDNS resolved '%s'", image)
					return a.renderImage(c, image, "/")
				}
			}
			if container, err := a.containerFromDomain(host); err == nil {
				pterm.Info.Printfln("magicDNS from dns domain resolved '%s'", container)
				return a.renderImage(c, container, "/")
			} else {
				pterm.Debug.Printfln("(magicDNS) failed getting records from TXT '%s'", err.Error())
			}

			return a.renderImage(c, a.defaultImage, "/")
		})

		ec.GET("/:registry/:org/:container/*", func(c echo.Context) error {
			org := c.Param("org")
			container := c.Param("container")
			registry := c.Param("registry")
			image := fmt.Sprintf("%s/%s/%s", registry, org, container)
			return a.renderImage(c, image, fmt.Sprintf("/%s/", image))
		})
	}
	return ec.Start(a.listenAddr)
}
