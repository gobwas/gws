package util

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func DumpRequestResponse(resp *http.Response) ([]byte, []byte, error) {
	if resp == nil {
		return nil, nil, fmt.Errorf("nil response")
	}

	if resp.Request == nil {
		return nil, nil, fmt.Errorf("nil request")
	}

	req, err := httputil.DumpRequest(resp.Request, true)
	if err != nil {
		return nil, nil, err
	}

	res, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return req, nil, err
	}

	return req, res, nil
}
