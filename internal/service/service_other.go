//go:build !darwin && !linux

package service

import (
	"errors"

	"github.com/aice109/ice-flow-cli/internal/i18n"
)

func install(string) (string, error) {
	return "", errors.New(i18n.T("background service is only supported on macOS and Linux"))
}

func uninstall() (string, error) {
	return "", errors.New(i18n.T("background service is only supported on macOS and Linux"))
}

func status() (string, error) {
	return i18n.T("unsupported on this platform"), nil
}
