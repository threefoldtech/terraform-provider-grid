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

type TwinResolver struct {
	cache  *cache.Cache
	client subi.SubstrateExt
}
type ProxyBus struct {
	signer      substrate.Identity
	endpoint    string
	twinID      uint32
	verifyReply bool
	resolver    TwinResolver
}

func NewProxyBus(endpoint string, twinID uint32, sub subi.SubstrateExt, signer substrate.Identity, verifyReply bool) (*ProxyBus, error) {
	if len(endpoint) != 0 && endpoint[len(endpoint)-1] == '/' {
		endpoint = endpoint[:len(endpoint)-1]
	}

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

func (r *ProxyBus) requestEndpoint(twinid uint32) string {
	return fmt.Sprintf("%s/twin/%d", r.endpoint, twinid)
}

func (r *ProxyBus) resultEndpoint(twinid uint32, retqueue string) string {
	return fmt.Sprintf("%s/twin/%d/%s", r.endpoint, twinid, retqueue)
}

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
		return errors.Wrap(err, "failed to serialize message")
	}
	resp, err := http.Post(r.requestEndpoint(twin), "application/json", bytes.NewBuffer(bs))
	if err != nil {
		return errors.Wrap(err, "error sending request")
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
		return errors.Wrap(err, "couldn't get twin public key")
	}
	if r.verifyReply {

		if err := msg.Verify(pk); err != nil {
			return err
		}
	}

	// errorred?
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
	msgDataBytes := getDecodedMsgData(msg.Data)

	if err := json.Unmarshal(msgDataBytes, result); err != nil {
		return errors.Wrap(err, "failed to decode response body")
	}

	return nil
}

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

func getDecodedMsgData(data string) []byte {
	decoded := []byte(data)
	b, err := base64.StdEncoding.DecodeString(data)
	if err == nil {
		decoded = b
	}
	return decoded
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
				errCount += 1
				err = lerr
				continue
			}
			if resp.StatusCode == 404 {
				// message not there yet
				continue
			}
			if resp.StatusCode != http.StatusOK {
				err = parseError(resp)
				errCount += 1
				continue
			}
			var msgs []rmb.Message
			if lerr := json.NewDecoder(resp.Body).Decode(&msgs); lerr != nil {
				err = lerr
				errCount += 1
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

type ErrorReply struct {
	Status  string `json:",omitempty"`
	Message string `json:",omitempty"`
	Error   string `json:"error"`
}

type ProxyResponse struct {
	Retqueue string
}
