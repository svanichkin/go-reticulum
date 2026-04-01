//go:build !linux

package ble

var _ = bluezIsDeviceBonded

func bluezIsDeviceBonded(_ string) (bool, error) { return true, nil }
