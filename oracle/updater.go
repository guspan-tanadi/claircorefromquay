package oracle

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/pkg/ovalutil"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const dbURL = `https://linux.oracle.com/security/oval/com.oracle.elsa-all.xml.bz2`

// Updater implements driver.Updater for Oracle Linux.
type Updater struct {
	ovalutil.Fetcher

	logger *zerolog.Logger // hack until the context-ified interfaces are used
}

// Option configures the provided Updater.
type Option func(*Updater) error

// NewUpdater returns an updater configured according to the provided Options.
func NewUpdater(opts ...Option) (*Updater, error) {
	u := Updater{}
	var err error
	u.Fetcher.URL, err = url.Parse(dbURL)
	if err != nil {
		return nil, err
	}
	u.Fetcher.Compression = ovalutil.CompressionBzip2
	for _, o := range opts {
		if err := o(&u); err != nil {
			return nil, err
		}
	}
	if u.logger == nil {
		u.logger = &log.Logger
	}
	l := u.logger.With().Str("component", u.Name()).Logger()
	u.logger = &l
	if u.Fetcher.Client == nil {
		u.Fetcher.Client = http.DefaultClient
	}

	return &u, nil
}

// WithClient returns an Option that will make the Updater use the specified
// http.Client, instead of http.DefaultClient.
func WithClient(c *http.Client) Option {
	return func(u *Updater) error {
		u.Fetcher.Client = c
		return nil
	}
}

// WithURL overrides the default URL to fetch an OVAL database.
func WithURL(uri, compression string) Option {
	c, cerr := ovalutil.ParseCompressor(compression)
	u, uerr := url.Parse(uri)
	return func(up *Updater) error {
		// Return any errors from the outer function.
		switch {
		case cerr != nil:
			return cerr
		case uerr != nil:
			return uerr
		}
		up.Fetcher.Compression = c
		up.Fetcher.URL = u
		return nil
	}
}

// WithLogger sets the default logger.
//
// Functions that take a context.Context will use the logger embedded in there
// instead of the Logger passed in via this Option.
func WithLogger(l *zerolog.Logger) Option {
	return func(u *Updater) error {
		u.logger = l
		return nil
	}
}

var _ driver.Updater = (*Updater)(nil)
var _ driver.FetcherNG = (*Updater)(nil)

// Name satifies the driver.Updater interface.
func (u *Updater) Name() string {
	return "oracle-updater"
}

// Fetch satifies the driver.Updater interface.
func (u *Updater) Fetch() (io.ReadCloser, string, error) {
	ctx := u.logger.WithContext(context.Background())
	ctx, done := context.WithTimeout(ctx, time.Minute)
	defer done()
	r, hint, err := u.FetchContext(ctx, "")
	return r, string(hint), err
}