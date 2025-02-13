// Copyright (c) The Arribada initiative.
// Licensed under the MIT License.

package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrAlreadyMounted = errors.New("already mounted")

func MountSSD(label string, mntPoint string) error {
	if runtime.GOOS == "darwin" {
		log.Printf("skipping mounting on darwin")
		return nil
	}

	mountPoint, err := filepath.Abs(mntPoint)
	if err != nil {
		return err
	}

	data, err := os.ReadFile("/etc/mtab")
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		l := sc.Text()
		parts := strings.Fields(l)
		if len(parts) < 2 {
			return fmt.Errorf("unexpected number of parts in /etc/mtab entry: %v", l)
		}
		if parts[1] == mountPoint {
			return ErrAlreadyMounted
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}

	cmd := exec.Command("mount", "-t", "vfat", "-o", "rw", "-L", label, mountPoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %v", err, string(out))
	}

	return nil
}

type balenaName struct {
	DeviceName string `json:"deviceName,omitempty"`
	Status     string `json:"success,omitempty"`
}

const DefaultDeviceName = "i360-default"

func GetDeviceName() (string, error) {
	addr := os.Getenv("BALENA_SUPERVISOR_ADDRESS")
	key := os.Getenv("BALENA_SUPERVISOR_API_KEY")
	url := fmt.Sprintf("%s/v2/device/name?apikey=%s", addr, key)

	if key == "" {
		return "", fmt.Errorf("BALENA_SUPERVISOR_API_KEY not set")
	}

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var name balenaName
	if err := json.Unmarshal(data, &name); err != nil {
		return "", err
	}

	return name.DeviceName, nil
}

func MaybeWrap(newErr, oldErr error, msg string) error {
	newErr = fmt.Errorf("%s: %v", msg, newErr)
	if oldErr == nil {
		return newErr
	}
	return fmt.Errorf("%v, %w", newErr, oldErr)
}
