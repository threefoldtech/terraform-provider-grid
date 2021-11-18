package provider

import (
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/substrate-client"
)

// validateAccount checks the mnemonics is associated with an account with key type ed25519
func validateAccount(apiClient *apiClient) error {
	_, err := apiClient.sub.GetAccount(apiClient.identity)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}
	if err != nil { // Account not found
		funcs := map[string]func(string) (substrate.Identity, error){"ed25519": substrate.NewIdentityFromEd25519Phrase, "sr25519": substrate.NewIdentityFromSr25519Phrase}
		for keyType, f := range funcs {
			ident, err2 := f(apiClient.mnemonics)
			if err2 != nil { // shouldn't happen, return original error
				log.Printf("couldn't convert the mneomincs to %s key: %s", keyType, err2.Error())
				return err
			}
			_, err2 = apiClient.sub.GetAccount(ident)
			if err2 == nil { // found an identity with key type other than the provided
				return fmt.Errorf("found an account with %s key type and the same mnemonics, make sure you provided the correct key type", keyType)
			}
		}
		// didn't find an account with any key type
		return err
	}
	return nil
}

func validateRedis(apiClient *apiClient) error {
	errMsg := fmt.Sprintf("redis error. make sure rmb_redis_url is correct and there's a redis server listening there. rmb_redis_url: %s", apiClient.rmb_redis_url)
	cl, err := newRedisPool(apiClient.rmb_redis_url)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	c, err := cl.Dial()
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	c.Close()
	return nil
}

func validateYggdrasil(apiClient *apiClient) error {
	twin, err := apiClient.sub.GetTwin(apiClient.twin_id)
	if err != nil {
		return errors.Wrapf(err, "coudln't get twin %d from substrate", apiClient.twin_id)
	}
	ip := net.ParseIP(twin.IP)
	listenIP := twin.IP
	if ip != nil && ip.To4() == nil {
		// if it's ipv6 surround it with brackets
		// otherwise, keep as is (can be ipv4 or a domain (probably will fail later but we don't care))
		listenIP = fmt.Sprintf("[%s]", listenIP)
	}
	s, err := net.Listen("tcp", fmt.Sprintf("%s:0", listenIP))
	if err != nil {
		return errors.Wrapf(err, "couldn't listen on port. make sure the twin id is associated with a valid yggdrasil ip, twin id: %d, ygg ip: %s, err", apiClient.twin_id, twin.IP)
	}
	defer s.Close()
	port := s.Addr().(*net.TCPAddr).Port
	arrived := false
	go func() {
		c, err := s.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			return
		}
		arrived = true
		c.Close()
	}()
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", listenIP, port))
	if err != nil {
		return errors.Wrapf(err, "failed to connect to ip. make sure the twin id is associated with a valid yggdrasil ip, twin id: %d, ygg ip: %s, err", apiClient.twin_id, twin.IP)
	}
	c.Close()
	if !arrived {
		return errors.Wrapf(err, "sent request but didn't arrive to me. make sure the twin id is associated with a valid yggdrasil ip, twin id: %d, ygg ip: %s, err", apiClient.twin_id, twin.IP)
	}
	return nil
}

func validateRMB(apiClient *apiClient) error {
	if err := validateRedis(apiClient); err != nil {
		return err
	}
	if err := validateYggdrasil(apiClient); err != nil {
		return err
	}
	return nil
}

func validateRMBProxyServer(apiClient *apiClient) error {
	resp, err := http.Get(apiClient.rmb_proxy_url)
	if err != nil {
		return errors.Wrapf(err, "couldn't reach rmb proxy at %s", apiClient.rmb_proxy_url)
	}
	if resp.StatusCode != 200 {
		return errors.Wrapf(err, "rmb proxy at %s returned status code %d", apiClient.rmb_proxy_url, resp.StatusCode)
	}
	return nil
}

func validateRMBProxy(apiClient *apiClient) error {
	if err := validateRMBProxyServer(apiClient); err != nil {
		return err
	}
	return nil
}

func preValidate(apiClient *apiClient) error {
	if apiClient.use_rmb_proxy {
		return validateRMBProxy(apiClient)
	} else {
		return validateRMB(apiClient)
	}
}

func validateAccountMoneyForExtrinsics(apiClient *apiClient) error {
	acc, err := apiClient.sub.GetAccount(apiClient.identity)
	if err != nil && !errors.Is(err, substrate.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}
	log.Printf("money %d\n", acc.Data.Free)
	if acc.Data.Free.Cmp(big.NewInt(20000)) == -1 {
		return fmt.Errorf("account contains %s, min fee is 20000", acc.Data.Free)
	}
	return nil
}

func newRedisPool(address string) (*redis.Pool, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var host string
	switch u.Scheme {
	case "tcp":
		host = u.Host
	case "unix":
		host = u.Path
	default:
		return nil, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}
	var opts []redis.DialOption

	if u.User != nil {
		opts = append(
			opts,
			redis.DialPassword(u.User.Username()),
		)
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(u.Scheme, host, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				//only check connection if more than 10 second of inactivity
				_, err := c.Do("PING")
				return err
			}

			return nil
		},
		MaxActive:   5,
		MaxIdle:     3,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}
