// Package download has utilities to download resources from URLs.
package download

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// Download downloads a file from a URL and writes it to path. If the url ends in .gz, .bz2, or .xz,
// it will be decompressed before writing.
func Download(client *http.Client, u *url.URL, path string) error {

	// atomically write to file
	dir, file := filepath.Split(path)
	if dir == "" {
		// If the file is in the current working directory, then dir will be "".
		// However, this means that ioutil.TempFile will use the default directory
		// for temporary files, which is wrong.
		dir = "."
	}

	// ensure dir exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(dir, file)
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer tmpFile.Close() // ignore err from closing twice

	// Clean up tmp file if not moved
	moved := false
	defer func() {
		if !moved {
			os.Remove(tmpFile.Name())
		}
	}()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := io.Reader(resp.Body)

	// decompress (optional)
	switch {
	case strings.HasSuffix(u.Path, "gz"):
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		r = gr
	case strings.HasSuffix(u.Path, "bz2"):
		r = bzip2.NewReader(resp.Body)
	case strings.HasSuffix(u.Path, "xz"):
		xzr, err := xz.NewReader(resp.Body)
		if err != nil {
			return err
		}
		r = xzr
	default:
		// don't decompress
	}

	if _, err := io.Copy(tmpFile, r); err != nil {
		return err
	}

	// Writes are not synchronous. Handle errors from writes returned by Close.
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("write and close temporary file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return err
	}

	moved = true

	return nil
}

// Decompress downloads a file from a URL and writes it to path. If the url ends in .gz, .bz2, or .xz,
// it will be decompressed before writing.
func Decompress(client *http.Client, u *url.URL, path string) error {

	// atomically write to file
	dir, file := filepath.Split(path)
	if dir == "" {
		// If the file is in the current working directory, then dir will be "".
		// However, this means that ioutil.TempFile will use the default directory
		// for temporary files, which is wrong.
		dir = "."
	}

	// ensure dir exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(dir, file)
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer tmpFile.Close() // ignore err from closing twice

	// Clean up tmp file if not moved
	moved := false
	defer func() {
		if !moved {
			os.Remove(tmpFile.Name())
		}
	}()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := io.Reader(resp.Body)

	// decompress (optional)
	switch {
	case strings.HasSuffix(u.Path, "gz"):
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		r = gr
	case strings.HasSuffix(u.Path, "bz2"):
		r = bzip2.NewReader(resp.Body)
	case strings.HasSuffix(u.Path, "xz"):
		xzr, err := xz.NewReader(resp.Body)
		if err != nil {
			return err
		}
		r = xzr
	default:
		// don't decompress
	}

	if _, err := io.Copy(tmpFile, r); err != nil {
		return err
	}

	// Writes are not synchronous. Handle errors from writes returned by Close.
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("write and close temporary file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return err
	}

	moved = true

	return nil
}
