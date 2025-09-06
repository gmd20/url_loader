//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	ProxyAddr = "http://127.0.0.1:1080"
)

func EnableProxy(enable bool) {
	if enable {
		// 设置当前进程环境变量（影响本程序及后续启动的子进程）
		os.Setenv("http_proxy", ProxyAddr)
		os.Setenv("https_proxy", ProxyAddr)
		os.Setenv("HTTP_PROXY", ProxyAddr)
		os.Setenv("HTTPS_PROXY", ProxyAddr)
		os.Setenv("no_proxy", "localhost,127.0.0.1,::1")
		os.Setenv("NO_PROXY", "localhost,127.0.0.1,::1")

		fmt.Println("✅ 代理已启用（当前进程及子进程生效）:", ProxyAddr)

		// 可选：设置 GNOME 代理（如果在桌面环境）
		setGnomeProxy(true)

		// 可选：写入 ~/.bashrc 持久化（谨慎使用）
		// writeProxyToBashrc(true)

	} else {
		// 清除环境变量
		os.Unsetenv("http_proxy")
		os.Unsetenv("https_proxy")
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("HTTPS_PROXY")
		os.Unsetenv("no_proxy")
		os.Unsetenv("NO_PROXY")

		fmt.Println("🚫 代理已关闭（当前进程及子进程）")

		// 可选：重置 GNOME 代理
		setGnomeProxy(false)

		// 可选：从 ~/.bashrc 移除代理设置
		// writeProxyToBashrc(false)
	}
}

// 设置 GNOME 代理（仅在 GNOME 桌面环境下有效）
func setGnomeProxy(enable bool) {
	cmd := "gsettings"
	if enable {
		// 设置手动代理
		exec.Command(cmd, "set", "org.gnome.system.proxy", "mode", "manual").Run()
		exec.Command(cmd, "set", "org.gnome.system.proxy.http", "host", "127.0.0.1").Run()
		exec.Command(cmd, "set", "org.gnome.system.proxy.http", "port", "1080").Run()
		exec.Command(cmd, "set", "org.gnome.system.proxy.https", "host", "127.0.0.1").Run()
		exec.Command(cmd, "set", "org.gnome.system.proxy.https", "port", "1080").Run()
		fmt.Println("✅ GNOME 代理已设置")
	} else {
		// 恢复为无代理
		exec.Command(cmd, "set", "org.gnome.system.proxy", "mode", "none").Run()
		fmt.Println("✅ GNOME 代理已关闭")
	}
}

// 可选：写入 ~/.bashrc 实现持久化（谨慎使用，避免重复写入）
func writeProxyToBashrc(enable bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("❌ 获取用户目录失败:", err)
		return
	}
	bashrc := home + "/.bashrc"

	content, err := os.ReadFile(bashrc)
	if err != nil {
		fmt.Println("❌ 读取 ~/.bashrc 失败:", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	marker := "# Go Proxy Auto Generated"
	proxyLines := []string{
		marker,
		`export http_proxy="http://127.0.0.1:1080"`,
		`export https_proxy="http://127.0.0.1:1080"`,
		`export HTTP_PROXY="http://127.0.0.1:1080"`,
		`export HTTPS_PROXY="http://127.0.0.1:1080"`,
		`export no_proxy="localhost,127.0.0.1,::1"`,
		`export NO_PROXY="localhost,127.0.0.1,::1"`,
	}

	var newLines []string
	inProxyBlock := false

	for _, line := range lines {
		if line == marker {
			inProxyBlock = true
			continue // 跳过旧标记和旧代理行
		}
		if inProxyBlock && strings.HasPrefix(line, "export ") {
			continue // 跳过旧代理设置行
		}
		if inProxyBlock && line == "" {
			inProxyBlock = false
			continue
		}
		if !inProxyBlock {
			newLines = append(newLines, line)
		}
	}

	if enable {
		newLines = append(newLines, "")
		newLines = append(newLines, proxyLines...)
	}

	err = os.WriteFile(bashrc, []byte(strings.Join(newLines, "\n")), 0644)
	if err != nil {
		fmt.Println("❌ 写入 ~/.bashrc 失败:", err)
		return
	}

	fmt.Println("✅ 代理设置已写入 ~/.bashrc，请重新加载终端或执行: source ~/.bashrc")
}
