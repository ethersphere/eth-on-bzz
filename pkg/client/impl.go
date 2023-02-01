// Copyright 2023 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethersphere/bee/pkg/api"
	"github.com/ethersphere/bee/pkg/swarm"
)

const (
	apiVersion  = "v1"
	userAgent   = "eth-on-bzz"
	contentType = "application/json"

	headerImmutable = "Immutable"
	headerBatchID   = api.SwarmPostageBatchIdHeader
)

type client struct {
	baseURL    string
	httpClient *http.Client
}

type Config struct {
	NodeURL string
}

func NewClient(cfg Config) Client {
	baseURL := cfg.NodeURL + "/" + apiVersion + "/"

	return &client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
}

func (c *client) BuyStamp(
	ctx context.Context,
	amount *big.Int,
	depth uint8,
	immutable bool,
) (BuyStampResponse, error) {
	var resp BuyStampResponse

	h := http.Header{}
	h.Add(headerImmutable, fmt.Sprintf("%v", immutable))

	endpoint := c.makeEndpoint("stamps", amount.Text(10), fmt.Sprintf("%d", depth))

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
	endpoint := c.makeEndpoint("bytes")

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
	endpoint := c.makeEndpoint("bytes", addr.String())

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
	endpoint := c.makeEndpoint("chunks", addr.String())

	httpResp, err := c.doRequest(ctx, http.MethodGet, endpoint, header, nil)
	if err != nil {
		return nil, fmt.Errorf("download chunk request failed: %w", err)
	}

	return httpResp.Body, nil
}

func (c *client) UploadSOC(
	ctx context.Context,
	owner common.Address,
	id SocID,
	data []byte,
	signature SocSignature,
	batchID BatchID,
) (UploadSOCResponse, error) {
	var resp UploadSOCResponse

	h := http.Header{}
	h.Add(headerBatchID, string(batchID))

	dataReader := bytes.NewReader(data)
	endpoint := c.makeEndpoint("soc", hex.EncodeToString(owner.Bytes()), hex.EncodeToString(id))
	endpoint += "?sig=" + string(signature)

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

func (c *client) FeedGet(
	ctx context.Context,
	owner common.Address,
	topic Topic,
) (FeedGetResponse, error) {
	return FeedGetResponse{}, nil
}

func (c *client) makeEndpoint(parts ...string) string {
	return c.baseURL + strings.Join(parts, "/")
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
