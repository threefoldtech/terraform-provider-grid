package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/threefoldtech/go-rmb"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

const (
	errThreshold = 4 // return error after failed 4 polls
)

// TwinResolver is a resolver for the twin
type TwinResolver struct {
	cache  *cache.Cache
	client subi.SubstrateExt
}

// ProxyBus struct
type ProxyBus struct {
	signer      substrate.Identity
	endpoint    string
	twinID      uint32
	verifyReply bool
	resolver    TwinResolver
}

// NewProxyBus generates a new proxy bus
func NewProxyBus(endpoint string, twinID uint32, sub subi.SubstrateExt, signer substrate.Identity, verifyReply bool) (*ProxyBus, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	return &ProxyBus{
		signer,
		endpoint,
		twinID,
		verifyReply,
		TwinResolver{
			cache:  cache.New(time.Minute*5, time.Minute),
			client: sub,
		},
	}, nil
}

func (r *ProxyBus) requestEndpoint(twinID uint32) string {
	return fmt.Sprintf("%s/twin/%d", r.endpoint, twinID)
}

func (r *ProxyBus) resultEndpoint(twinID uint32, retqueue string) string {
	return fmt.Sprintf("%s/twin/%d/%s", r.endpoint, twinID, retqueue)
}

// Call calls a function via rmb
func (r *ProxyBus) Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize request data")
	}

	msg := rmb.Message{
		Version:    1,
		Expiration: 3600,
		Command:    fn,
		TwinSrc:    int(r.twinID),
		TwinDst:    []int{int(twin)},
		Data:       base64.StdEncoding.EncodeToString(bs),
		Epoch:      time.Now().Unix(),
		Proxy:      true,
	}
	if err := msg.Sign(r.signer); err != nil {
		return err
	}
	bs, err = json.Marshal(msg)
	if err != nil {
		return errors.Wrapf(err, "failed to serialize RMB message")
	}
	resp, err := http.Post(r.requestEndpoint(twin), "application/json", bytes.NewBuffer(bs))
	if err != nil {
		return errors.Wrapf(err, "error sending request for twin id (%d)", twin)
	}
	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}
	var res ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return errors.Wrap(err, "failed to decode proxy response body")
	}
	msg, err = r.pollResponse(ctx, twin, res.Retqueue)
	if err != nil {
		return errors.Wrap(err, "couldn't poll response")
	}
	pk, err := r.resolver.PublicKey(int(twin))
	if err != nil {
		return errors.Wrapf(err, "couldn't get twin public key for twin id (%d)", twin)
	}
	if r.verifyReply {

		if err := msg.Verify(pk); err != nil {
			return err
		}
	}

	// errored?
	if len(msg.Err) != 0 {
		return errors.New(msg.Err)
	}

	// not expecting a result
	if result == nil {
		return nil
	}

	if len(msg.Data) == 0 {
		return fmt.Errorf("no response body was returned")
	}

	//check if msg.Data is base64 encoded
	msgDataBytes := TryDecodeBase64OrElse(msg.Data)

	if err := json.Unmarshal(msgDataBytes, result); err != nil {
		return errors.Wrap(err, "failed to decode response body")
	}

	return nil
}

// PublicKey returns the public key for the twin
func (r TwinResolver) PublicKey(twin int) ([]byte, error) {
	key := fmt.Sprintf("pk:%d", twin)
	cached, ok := r.cache.Get(key)
	if ok {
		return cached.([]byte), nil
	}
	pk, err := r.client.GetTwinPK(uint32(twin))
	if err != nil {
		return nil, err
	}

	r.cache.Set(key, pk, cache.DefaultExpiration)
	return pk, nil
}

func (r *ProxyBus) pollResponse(ctx context.Context, twin uint32, retqueue string) (rmb.Message, error) {
	ts := time.NewTicker(1 * time.Second)
	errCount := 0
	var err error
	for {
		select {
		case <-ts.C:
			if errCount == errThreshold {
				return rmb.Message{}, err
			}
			resp, lerr := http.Get(r.resultEndpoint(twin, retqueue))
			if lerr != nil {
				log.Printf("failed to send result-fetching request: %s", err.Error())
				errCount++
				err = lerr
				continue
			}
			if resp.StatusCode == 404 {
				// message not there yet
				continue
			}
			if resp.StatusCode != http.StatusOK {
				err = parseError(resp)
				errCount++
				continue
			}
			var msgs []rmb.Message
			if lerr := json.NewDecoder(resp.Body).Decode(&msgs); lerr != nil {
				err = lerr
				errCount++
				continue
			}
			if len(msgs) == 0 {
				// nothing there yet
				continue
			}
			return msgs[0], nil
		case <-ctx.Done():
			return rmb.Message{}, errors.New("context cancelled")
		}
	}
}

func parseError(resp *http.Response) error {
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read error response (%d)", resp.StatusCode)
	}

	var errContent ErrorReply
	if err := json.Unmarshal(bodyBytes, &errContent); err != nil {
		return fmt.Errorf("(%d): %s", resp.StatusCode, string(bodyBytes))
	}

	// just in case it was decided to unify grid proxy error messages
	if errContent.Error != "" {
		return fmt.Errorf("(%d): %s", resp.StatusCode, errContent.Error)
	}

	if errContent.Message != "" {
		return fmt.Errorf("(%d): %s", resp.StatusCode, errContent.Message)
	}

	return fmt.Errorf("%s (%d)", http.StatusText(resp.StatusCode), resp.StatusCode)
}

// ErrorReply struct for error response
type ErrorReply struct {
	Status  string `json:",omitempty"`
	Message string `json:",omitempty"`
	Error   string `json:"error"`
}

// ProxyResponse struct for proxy response
type ProxyResponse struct {
	Retqueue string
}
