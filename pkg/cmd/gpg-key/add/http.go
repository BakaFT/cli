package add

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghinstance"
)

var scopesError = errors.New("insufficient OAuth scopes")

func gpgKeyUpload(httpClient *http.Client, hostname string, keyFile io.Reader) error {
	if !isGpgKeyArmored(keyFile) {
		return errors.New("invalid GPG key")
	}
	url := ghinstance.RESTPrefix(hostname) + "user/gpg_keys"

	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		return err
	}

	payload := map[string]string{
		"armored_public_key": string(keyBytes),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)

	// fmt.Println(string(body))

	if resp.StatusCode == 404 {
		return scopesError
	} else if resp.StatusCode > 299 {
		var httpError api.HTTPError
		err := api.HandleHTTPError(resp)
		if errors.As(err, &httpError) && isDuplicateError(&httpError) {
			return nil
		}
		return err
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func isDuplicateError(err *api.HTTPError) bool {
	return err.StatusCode == 422 && len(err.Errors) == 2 &&
		err.Errors[0].Field == "key_id" && err.Errors[0].Message == "key_id already exists"
}

func isGpgKeyArmored(keyFile io.Reader) bool {
	buf := make([]byte, 36)
	_, err := keyFile.Read(buf)
	if err != nil {
		return false
	}
	return bytes.Equal(buf, []byte("-----BEGIN PGP PUBLIC KEY BLOCK-----"))

}
