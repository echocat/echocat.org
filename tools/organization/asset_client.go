package main

import (
	"flag"
	"fmt"
	log "github.com/echocat/slf4g"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var (
	assetsFolder = flag.String("assets", "assets", "Folder to put downloaded assets to.")
)

type assetClient struct {
	cache map[string]string
	mutex sync.Mutex
}

func newAssetClient() *assetClient {
	return &assetClient{
		cache: make(map[string]string),
	}
}

func (instance *assetClient) retrieve(sourceUrl string) (resultAsset string, err error) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	l := log.With("url", sourceUrl)

	defer func() {
		if err == nil {
			l.With("file", resultAsset).Info("Asset downloaded.")
		}
	}()

	if cached := instance.cache[sourceUrl]; cached != "" {
		return cached, nil
	}

	resp, err := http.Get(sourceUrl)
	if err != nil {
		return "", fmt.Errorf("cannot download '%s': %w", sourceUrl, err)
	}
	if resp.Body == nil {
		return "", fmt.Errorf("no response body while try to  download '%s': %d - %s", sourceUrl, resp.StatusCode, resp.Status)
	}
	defer func() {
		if cErr := resp.Body.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status while try to  download '%s': %d - %s", sourceUrl, resp.StatusCode, resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return "", fmt.Errorf("no Content-Type header present try to  download '%s': %d - %s", sourceUrl, resp.StatusCode, resp.Status)
	}
	ext, err := instance.contentTypeToExt(contentType, sourceUrl)
	if err != nil {
		return "", err
	}

	return instance.retrieveFromReader(resp.Body, sourceUrl, ext)
}

func (instance *assetClient) contentTypeToExt(contentType, sourceUrl string) (string, error) {
	ext, err := mime.ExtensionsByType(contentType)
	if err != nil {
		return "", fmt.Errorf("illegal Content-Type header present try to  download '%s': %s - %w", sourceUrl, contentType, err)
	}
	if len(ext) <= 0 {
		return "", fmt.Errorf("illegal Content-Type header present try to  download '%s': %s", sourceUrl, contentType)
	}

	for _, candidate := range ext {
		if candidate == ".jpg" {
			return candidate, nil
		}
	}

	return ext[0], nil
}

func (instance *assetClient) retrieveFromReader(source io.Reader, sourceRef, ext string) (string, error) {
	if err := os.MkdirAll(*assetsFolder, 755); err != nil {
		return "", fmt.Errorf("cannot create target folder '%s' to store the '%s' inside: %w", *assetsFolder, sourceRef, err)
	}

	w, err := ioutil.TempFile(*assetsFolder, "~*"+ext)
	if err != nil {
		return "", fmt.Errorf("cannot temporary file: %w", err)
	}
	defer func() {
		_ = w.Close()
	}()

	r := newSha256reader(source)
	if _, err = io.Copy(w, r); err != nil {
		return "", fmt.Errorf("cannot download '%s' to '%s': %w", sourceRef, w.Name(), err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("cannot closed '%s' after downloaded from '%s': %w", w.Name(), sourceRef, err)
	}

	target := filepath.Join(*assetsFolder, fmt.Sprintf("%s%s", r.SumString(), ext))

	if err := os.Rename(w.Name(), target); err != nil {
		return "", fmt.Errorf("cannot rename '%s' to '%s' after downloaded from '%s': %w", w.Name(), target, sourceRef, err)
	}

	return filepath.Base(target), nil
}

func (instance *assetClient) cleanTarget() error {
	if err := os.RemoveAll(*assetsFolder); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
