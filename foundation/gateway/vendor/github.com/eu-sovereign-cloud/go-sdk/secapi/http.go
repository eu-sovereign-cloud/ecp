package secapi

import (
	"fmt"
	"net/http"
)

type httpResponse interface {
	StatusCode() int
}

func checkStatusCode(resp httpResponse, alloweds ...int) error {
	code := resp.StatusCode()
	for _, allowed := range alloweds {
		if code == allowed {
			return nil
		}
	}
	return fmt.Errorf("unexpected status code %d", code)
}

func checkSuccessPutStatusCodes(resp httpResponse) error {
	if err := checkStatusCode(resp, http.StatusOK, http.StatusCreated); err != nil {
		return err
	}
	return nil
}

func checkSuccessPostStatusCodes(resp httpResponse) error {
	if err := checkStatusCode(resp, http.StatusAccepted); err != nil {
		return err
	}
	return nil
}

func checkSuccessDeleteStatusCodes(resp httpResponse) error {
	if err := checkStatusCode(resp, http.StatusAccepted, http.StatusNotFound); err != nil {
		return err
	}
	return nil
}
