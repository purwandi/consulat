package consulat

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/lib/file"
)

const (
	durationLease = 60 * time.Second
)

type Consulat struct {
	Client      *api.Client
	ACLToken    *api.ACLToken
	AuthMethod  string
	BearerToken string
	TokenFile   string
}

func New(authMethod, jwtFile, tokenFile string) (*Consulat, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(jwtFile)
	if err != nil {
		return nil, err
	}

	bearerToken := strings.TrimSpace(string(data))
	if bearerToken == "" {
		return nil, fmt.Errorf("no bearer token found in %s", jwtFile)
	}

	if tokenFile == "" {
		return nil, fmt.Errorf("no token found in %s", tokenFile)
	}

	return &Consulat{
		Client:      client,
		AuthMethod:  authMethod,
		BearerToken: bearerToken,
		TokenFile:   tokenFile,
	}, nil
}

func (c *Consulat) Login() error {
	var err error

	params := &api.ACLLoginParams{
		AuthMethod:  c.AuthMethod,
		BearerToken: c.BearerToken,
	}

	c.ACLToken, _, err = c.Client.ACL().Login(params, nil)
	if err != nil {
		return err
	}

	payload := []byte(c.ACLToken.SecretID)
	file.WriteAtomicWithPerms(c.TokenFile, payload, 0o755, 0o600)

	log.Println("successfully generate token")
	return nil
}

func (c *Consulat) durationLease() time.Duration {
	now := time.Now()
	duration := c.ACLToken.ExpirationTime.Sub(now)

	return duration - durationLease
}

func (c *Consulat) Renew() error {
	for {
		// renew token before expired
		duration := c.durationLease()
		log.Println("renewing until: ", duration)
		time.Sleep(duration)

		log.Println("renewing credentials:")
		if err := c.Login(); err != nil {
			log.Println(err)
			continue
		}
	}
}

func (c *Consulat) Logout() error {
	params := &api.WriteOptions{
		Token: c.ACLToken.SecretID,
	}
	_, err := c.Client.ACL().Logout(params)
	return err
}
