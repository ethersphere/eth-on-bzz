// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethersphere/bee/pkg/api"
	"github.com/ethersphere/bee/pkg/swarm"
)

const (
	apiVersion  = "v1"
	userAgent   = "eth-on-bzz"
	contentType = "application/json"

	headerImmutable        = "Immutable"
	headerBatchID          = api.SwarmPostageBatchIdHeader
	headerFeedCurrentIndex = api.SwarmFeedIndexHeader
	headerFeedNextIndex    = api.SwarmFeedIndexNextHeader

	portAPI  = 1633
	portAPId = 1635
)

type client struct {
	cfg        Config
	httpClient *http.Client
}

type Config struct {
	NodeURL string
}

func NewClient(cfg Config) Client {
	return &client{
		cfg:        cfg,
		httpClient: http.DefaultClient,
	}
}

func (c *client) Stamps(
	ctx context.Context,
) (StampsResponse, error) {
	var resp StampsResponse

	h := http.Header{}

	endpoint := c.makeEndpoint(portAPId, "stamps")

	//nolint:bodyclose // body is closed after handling error
	httpResp, err := c.doRequest(ctx, http.MethodGet, endpoint, h, nil)
	if err != nil {
		return resp, fmt.Errorf("stamps request failed: %w", err)
	}

	defer closeBody(httpResp)

	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return resp, fmt.Errorf("failed to decode response from stamps endpoint: %w", err)
	}

	return resp, nil
}

func (c *client) BuyStamp(
	ctx context.Context,
	amount *big.Int,
	depth uint8,
	immutable bool,
) (BuyStampResponse, error) {
	var resp BuyStampResponse

	h := http.Header{}
	h.Add(headerImmutable, strconv.FormatBool(immutable))

	endpoint := c.makeEndpoint(portAPId, "stamps", amount.Text(10), strconv.Itoa(int(depth)))

	//nolint:bodyclose // body is closed after handling error
	httpResp, err := c.doRequest(ctx, http.MethodPost, endpoint, h, nil)
	if err != nil {
		return resp, fmt.Errorf("buy stamps request failed: %w", err)
	}

	defer closeBody(httpResp)

	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return resp, fmt.Errorf("failed to decode response from buy stamps endpoint: %w", err)
	}

	return resp, nil
}

func (c *client) Upload(
	ctx context.Context,
	data []byte,
	batchID BatchID,
) (UploadResponse, error) {
	var resp UploadResponse

	h := http.Header{}
	h.Add(headerBatchID, string(batchID))

	dataReader := bytes.NewReader(data)
	endpoint := c.makeEndpoint(portAPI, "bytes")

	//nolint:bodyclose // body is closed after handling error
	httpResp, err := c.doRequest(ctx, http.MethodPost, endpoint, h, dataReader)
	if err != nil {
		return resp, fmt.Errorf("upload request failed: %w", err)
	}

	defer closeBody(httpResp)

	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return resp, fmt.Errorf("failed to decode response from upload endpoint: %w", err)
	}

	return resp, nil
}

func (c *client) Download(
	ctx context.Context,
	addr swarm.Address,
) (io.ReadCloser, error) {
	header := http.Header{}
	endpoint := c.makeEndpoint(portAPI, "bytes", addr.String())

	httpResp, err := c.doRequest(ctx, http.MethodGet, endpoint, header, nil)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}

	return httpResp.Body, nil
}

func (c *client) DownloadChunk(
	ctx context.Context,
	addr swarm.Address,
) (io.ReadCloser, error) {
	header := http.Header{}
	endpoint := c.makeEndpoint(portAPI, "chunks", addr.String())

	httpResp, err := c.doRequest(ctx, http.MethodGet, endpoint, header, nil)
	if err != nil {
		return nil, fmt.Errorf("download chunk request failed: %w", err)
	}

	return httpResp.Body, nil
}

func (c *client) UploadSoc(
	ctx context.Context,
	owner common.Address,
	id SocID,
	data []byte,
	signature SocSignature,
	batchID BatchID,
) (UploadSocResponse, error) {
	var resp UploadSocResponse

	h := http.Header{}
	h.Add(headerBatchID, string(batchID))

	dataReader := bytes.NewReader(data)

	ownerParam := hex.EncodeToString(owner.Bytes())
	idParam := hex.EncodeToString(id)
	endpoint := c.makeEndpoint(portAPI, "soc", ownerParam, idParam)
	endpoint += "?sig=" + hex.EncodeToString(signature)

	//nolint:bodyclose // body is closed after handling error
	httpResp, err := c.doRequest(ctx, http.MethodPost, endpoint, h, dataReader)
	if err != nil {
		return resp, fmt.Errorf("upload request failed: %w", err)
	}

	defer closeBody(httpResp)

	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return resp, fmt.Errorf("failed to decode response from upload SOC endpoint: %w", err)
	}

	return resp, nil
}

func (c *client) FeedIndexLatest(
	ctx context.Context,
	owner common.Address,
	topic Topic,
) (FeedIndexResponse, error) {
	resp := FeedIndexResponse{}

	h := http.Header{}

	ownerParam := hex.EncodeToString(owner.Bytes())
	topicParam := hex.EncodeToString(topic)
	endpoint := c.makeEndpoint(portAPI, "feeds", ownerParam, topicParam)

	//nolint:bodyclose // body is closed after handling error
	httpResp, err := c.doRequest(ctx, http.MethodGet, endpoint, h, nil)
	if err != nil {
		return resp, fmt.Errorf("feeds request failed: %w", err)
	}

	defer closeBody(httpResp)

	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return resp, fmt.Errorf("failed to decode response from feeds endpoint: %w", err)
	}

	resp.Current, err = decodeIndexFromHeader(httpResp.Header.Get(headerFeedCurrentIndex))
	if err != nil {
		return resp, fmt.Errorf("failed to decode header data from feeds endpoint: %w", err)
	}

	resp.Next, err = decodeIndexFromHeader(httpResp.Header.Get(headerFeedNextIndex))
	if err != nil {
		return resp, fmt.Errorf("failed to decode header data from feeds endpoint: %w", err)
	}

	return resp, nil
}

//nolint:wrapcheck //relax
func decodeIndexFromHeader(val string) (uint64, error) {
	ds, err := hex.DecodeString(val)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(ds), nil
}

func (c *client) makeEndpoint(port int, parts ...string) string {
	resource := strings.Join(parts, "/")

	return c.cfg.NodeURL + ":" + strconv.Itoa(port) + "/" + apiVersion + "/" + resource
}

func (c *client) doRequest(
	ctx context.Context,
	method, path string,
	header http.Header,
	body io.Reader,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header = header
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		closeBody(resp)

		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	if err := responseErrorHandler(resp); err != nil {
		closeBody(resp)

		return nil, err
	}

	return resp, nil
}

type swarmAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e swarmAPIError) Error() string {
	return fmt.Sprintf("api error: code %d, message: %v", e.Code, e.Message)
}

func responseErrorHandler(r *http.Response) error {
	if r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	var eResp swarmAPIError
	if err := json.NewDecoder(r.Body).Decode(&eResp); err != nil {
		return fmt.Errorf("failed to decode swarm api error response: %w", err)
	}

	return eResp
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}
