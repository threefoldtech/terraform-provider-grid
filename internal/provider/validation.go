package provider

import (
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

// validateAccount checks the mnemonics is associated with an account with key type ed25519
func validateAccount(apiClient *apiClient, sub subi.SubstrateExt) error {
	_, err := sub.GetAccount(apiClient.identity)
	if err != nil && !errors.Is(err, subi.ErrAccountNotFound) {
		return errors.Wrap(err, "failed to get account with the given mnemonics")
	}
	if err != nil { // Account not found
		funcs := map[string]func(string) (subi.Identity, error){"ed25519": subi.NewIdentityFromEd25519Phrase, "sr25519": subi.NewIdentityFromSr25519Phrase}
		for keyType, f := range funcs {
			ident, err2 := f(apiClient.mnemonics)
			if err2 != nil { // shouldn't happen, return original error
				log.Printf("couldn't convert the mnemomincs to %s key: %s", keyType, err2.Error())
				return err
			}
			_, err2 = sub.GetAccount(ident)
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

func validateYggdrasil(apiClient *apiClient, sub subi.SubstrateExt) error {
	yggIP, err := sub.GetTwinIP(apiClient.twin_id)
	if err != nil {
		return errors.Wrapf(err, "coudln't get twin %d from substrate", apiClient.twin_id)
	}
	ip := net.ParseIP(yggIP)
	listenIP := yggIP
	if ip != nil && ip.To4() == nil {
		// if it's ipv6 surround it with brackets
		// otherwise, keep as is (can be ipv4 or a domain (probably will fail later but we don't care))
		listenIP = fmt.Sprintf("[%s]", listenIP)
	}
	s, err := net.Listen("tcp", fmt.Sprintf("%s:0", listenIP))
	if err != nil {
		return errors.Wrapf(err, "couldn't listen on port. make sure the twin id is associated with a valid yggdrasil ip, twin id: %d, ygg ip: %s, err", apiClient.twin_id, yggIP)
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
		return errors.Wrapf(err, "failed to connect to ip. make sure the twin id is associated with a valid yggdrasil ip, twin id: %d, ygg ip: %s, err", apiClient.twin_id, yggIP)
	}
	c.Close()
	if !arrived {
		return errors.Wrapf(err, "sent request but didn't arrive to me. make sure the twin id is associated with a valid yggdrasil ip, twin id: %d, ygg ip: %s, err", apiClient.twin_id, yggIP)
	}
	return nil
}

func validateRMB(apiClient *apiClient, sub subi.SubstrateExt) error {
	if err := validateRedis(apiClient); err != nil {
		return err
	}
	if err := validateYggdrasil(apiClient, sub); err != nil {
		return err
	}
	return nil
}

func validateRMBProxyServer(apiClient *apiClient) error {
	return apiClient.grid_client.Ping()
}

func validateRMBProxy(apiClient *apiClient) error {
	if err := validateRMBProxyServer(apiClient); err != nil {
		return err
	}
	return nil
}

func preValidate(apiClient *apiClient, sub subi.SubstrateExt) error {
	if apiClient.use_rmb_proxy {
		return validateRMBProxy(apiClient)
	} else {
		return validateRMB(apiClient, sub)
	}
}

func validateAccountMoneyForExtrinsics(sub subi.SubstrateExt, identity subi.Identity) error {
	acc, err := sub.GetAccount(identity)
	if err != nil && !errors.Is(err, subi.ErrAccountNotFound) {
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
