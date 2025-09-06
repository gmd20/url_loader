//go:build windows
// +build windows

package main

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
)

func EnableProxy(enable bool) {
	var proxyEnable uint32 = 0
	if enable {
		proxyEnable = 1
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		fmt.Println("Failed to open registry key", err)
		return
	}
	defer key.Close()

	err = key.SetDWordValue("ProxyEnable", proxyEnable)
	if err != nil {
		fmt.Println("Failed to set registry ProxyEnable", err)
		return
	}

	/*
		  err = key.SetStringValue("ProxyServer", "http://127.0.0.1:8000")
			if err != nil {
				fmt.Println("Failed to set registryProxyServer", err)
		    return
			}
	*/
}
