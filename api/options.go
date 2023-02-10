package api

import (
	units "github.com/docker/go-units"
	"github.com/moby/moby/api/types"
	"github.com/mudler/containerbay/store"
	str2duration "github.com/xhit/go-str2duration/v2"
)

// Options is a generic handler which mutates an API object
type Options func(a *API) error

// WithDNSField sets the DNS TXT fields which containerbay will use
// to resolve the container image to host the website
func WithDNSField(s string) func(*API) error {
	return func(a *API) error {
		a.dnsTXTKey = s
		return nil
	}
}

// WithWhitelist adds a list of regexes to be whiteliste of the images that can be served.
// Those that doesn't match regexes are refused
func WithWhitelist(s ...string) func(*API) error {
	return func(a *API) error {
		a.whitelist = append(a.whitelist, s...)
		return nil
	}
}

// WithDefaultImage sets the DefaultImage which will be used
// as fallback
func WithDefaultImage(s string) func(*API) error {
	return func(a *API) error {
		a.defaultImage = s
		return nil
	}
}

// WithListeningAddress sets the API listening address port
// in the ip:port form. To bind to all IPs, just set the port ( e.g. ":8080" )
func WithListeningAddress(s string) func(*API) error {
	return func(a *API) error {
		a.listenAddr = s
		return nil
	}
}

// WithMagicDNS sets the magic dns domain used to query the images from
// e.g. to allow requests like http://registry.org.image.tag.magicdns
func WithMagicDNS(s string) func(*API) error {
	return func(a *API) error {
		a.magicDNS = s
		return nil
	}
}

// WithCacheStore sets the cache store where all container images will be stored
func WithCacheStore(s string) func(*API) error {
	return func(a *API) error {
		a.cacheStore = store.New(s)
		return a.cacheStore.EnsureExists()
	}
}

// WithCleanupInterval sets the cleanup interval that triggers a cache cleanup
// The cache gets periodically cleaned up at the interval provided from the images contained in it
// The interval is a string and specifies a duration, e.g. 10m, 2h, 12h
func WithCleanupInterval(s string) func(*API) error {
	return func(a *API) error {
		durationFromString, err := str2duration.ParseDuration(s)
		if err != nil {
			return err
		}
		a.cleanupInterval = durationFromString
		return nil
	}
}

// WithMaxSize specifies a max size of the images to be served. Images bigger
// than the specified size are not served and an error to the client is returned
// Valid values are e.g. 10MB, 2GB, etc.
func WithMaxSize(s string) func(*API) error {
	return func(a *API) error {
		size, err := units.FromHumanSize(s)
		if err != nil {
			return err
		}
		a.maxSize = size
		return nil
	}
}

// WithPoolSize specify a size for the queue of the worker pool
func WithPoolSize(i int) func(*API) error {
	return func(a *API) error {
		a.poolSize = i
		return nil
	}
}

// WithWorkers specifies the number of running workers
// that will download images in parallel
func WithWorkers(i int) func(*API) error {
	return func(a *API) error {
		a.workers = i
		return nil
	}
}

// WithAuth specify an authentication which is used to perform
// image pulls.
// This is mostly required to access to private registries or either
// workaround Pull rate limits
func WithAuth(auth *types.AuthConfig) func(*API) error {
	return func(a *API) error {
		// auth := &types.AuthConfig{
		// 	Username:      user,
		// 	Password:      pass,
		// 	ServerAddress: server,
		// 	Auth:          authType,
		// 	IdentityToken: identity,
		// 	RegistryToken: registryToken,
		// }
		a.auth = auth
		return nil
	}
}

// Standalone sets the standalone image to serve requests from.
func Standalone(image string) func(*API) error {
	return func(a *API) error {
		a.standaloneImage = image
		return nil
	}
}

// New returns a new API instance with the given options
func New(opts ...Options) *API {
	a := &API{
		workers:   1,
		dnsTXTKey: "containerbay",
	}
	for _, o := range opts {
		o(a)
	}
	return a
}
