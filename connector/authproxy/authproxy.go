// Package authproxy implements a connector which relies on external
// authentication (e.g. mod_auth in Apache2) and returns an identity with the
// HTTP header X-Remote-User as verified email.
package authproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

// Config holds the configuration parameters for a connector which returns an
// identity with the HTTP header X-Remote-User as verified email.
type Config struct {
	UserHeader string `json:"userHeader"`
}

// Open returns an authentication strategy which requires no user interaction.
func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	userHeader := c.UserHeader
	if userHeader == "" {
		userHeader = "X-Remote-User"
	}

	return &callback{userHeader: userHeader, logger: logger, pathSuffix: "/" + id}, nil
}

// Callback is a connector which returns an identity with the HTTP header
// X-Remote-User as verified email.
type callback struct {
	userHeader string
	logger     log.Logger
	pathSuffix string
}

// LoginURL returns the URL to redirect the user to login with.
func (m *callback) LoginURL(s connector.Scopes, callbackURL, state string) (string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse callbackURL %q: %v", callbackURL, err)
	}
	u.Path += m.pathSuffix
	v := u.Query()
	v.Set("state", state)
	u.RawQuery = v.Encode()
	return u.String(), nil
}

// HandleCallback parses the request and returns the user's identity
func (m *callback) HandleCallback(s connector.Scopes, r *http.Request) (connector.Identity, error) {
	m.logger.Debugf("Headers: %v", r.Header)
	remoteUser := r.Header.Get(m.userHeader)
	if remoteUser == "" {
		return connector.Identity{}, fmt.Errorf("need login redirect")
	}

	identity := connector.Identity{
		UserID: remoteUser,
	}

	displayName := r.Header.Get("X-Shib-displayName")
	if displayName != "" {
		identity.Username = displayName
	}

	eppn := r.Header.Get("X-Shib-eduPersonPrincipalName")
	if eppn != "" {
		identity.PreferredUsername = eppn
	}

	shibMail := r.Header.Get("X-Shib-mail")
	if shibMail != "" {
		identity.Email = shibMail
		identity.EmailVerified = true
	}

	shibAffiliation := r.Header.Get("X-Shib-eduPersonScopedAffiliation")
	if shibAffiliation != "" {
		identity.Groups = strings.Split(shibAffiliation, ";")
	}

	return identity, nil
}
