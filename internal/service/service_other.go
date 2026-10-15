//go:build !darwin && !linux

package service

import "fmt"

func install(string) (string, error) {
	return "", fmt.Errorf("background service is only supported on macOS and Linux")
}

func uninstall() (string, error) {
	return "", fmt.Errorf("background service is only supported on macOS and Linux")
}

func status() (string, error) {
	return "unsupported on this platform", nil
}
