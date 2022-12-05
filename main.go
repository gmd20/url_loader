package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	neturl "net/url"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"golang.org/x/sys/windows/registry"
)

// 配置参考
// https://github.com/xtls/Xray-examples
// https://xtls.github.io/Xray-docs-next/config/outbounds/trojan.html#serverobject
// https://github.com/v2fly/v2ray-examples
const configTemplate string = `
    { {{if .ps }} // {{.ps}} {{end}}
      "protocol": "{{.protocol}}",  // 出口协议
      "tag": "Proxy",
{{if eq .protocol "shadowsocks" }}
      "settings": {
        "servers": [
          {
            "address": "{{.address}}", // Shadowsocks 的服务器地址
            "method": "{{.method}}", // Shadowsocks 的加密方式
             // "ota": true, // 是否开启 OTA，true 为开启
             "password": "{{.password}}", // Shadowsocks 的密码
             "port": {{.port}}
          }
        ]
      },
      "streamSettings": {
        "network": "tcp"
      }
{{else if eq .protocol "trojan" }}
      "settings": {
        "servers": [
         {
           "address": "{{.address}}",
           "password": "{{.password}}",
{{if .flow }}
           "flow": "{{.flow}}",
{{end}}
           "port": {{.port}}
          }
        ]
      },
      "streamSettings": {
{{if .security }}
        "security": "{{.security}}",
{{end}}
{{if eq .security "tls"}}
        "tlsSettings": {
{{if .sni }}
          "serverName": "{{.sni}}" // 换成你的域名
{{end}}
        },
{{end}}
{{if eq .security "xtls"}}
        "xtlsSettings": {
{{if .sni }}
            "serverName": "{{.sni}}"  // 你的域名
{{end}}
        },
{{end}}
        "network": "tcp"
     }
{{else}}
      "settings": {
        "vnext": [
          {
            "address": "{{.address}}", // 服务器地址，请修改为你自己的服务器 IP 或域名
            "port": {{.port}},         // 服务器端口
            "users": [
              {
{{if .alterId }}
                "alterId": {{.alterId}}, // 此处的值也应当与服务器相同
{{end}}
{{if eq .protocol "vless"}}
{{if .encryption }}
                "encryption": "{{.encryption}}",
{{else}}
                "encryption": "none",
{{end}}
                "level": 0,
{{else if eq .protocol "vmess"}}
{{if and (eq .network "ws")  (eq .security "tls")}}
                "security": "none",
                "level": 0,
{{end}}
{{end}}
{{if .flow }}
                "flow": "{{.flow}}"
{{end}}
                "id": "{{.id}}"  // 用户 ID，必须与服务器端配置相同

              }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "{{.network}}",  // network: "tcp" | "kcp" | "ws" | "http" | "domainsocket" | "quic"| "xtls"
{{if .security }}
        "security": "{{.security}}",
{{end}}
        "tlsSettings": {
{{if eq .type "http" }}
          "header": {
              "type": "http",
              "request": {
                  "path": [
                      "{{.path}}" // 必须换成自定义的 PATH，需要和服务端的一致
                  ]
              }
          },
{{end}}
{{if .sni }}
          "serverName": "{{.sni}}" // 换成你的域名
{{else if .host }}
          "serverName": "{{.host}}" // 换成你的域名
{{end}}


        },
        "xtlsSettings": {
{{if .sni }}
            "serverName": "{{.sni}}"  // 你的域名
{{end}}
        },
        "wsSettings": { // websocket 设置
{{if ne .host ""}}
           "headers": {
             "Host": "{{.host}}"
           },
{{end}}
{{if .path }}
          "path": "{{.path}}" // 与nginx配置相关
{{end}}
        },
        "httpSettings": { // 此项是关于 HTTP/2 的设置
{{if .path }}
          "path": "{{.path}}"
{{end}}
        }
      }
{{end}}
    },
`

func readUrl() string {
	var cmd string
	fmt.Printf("input url>")
	reader := bufio.NewReader(os.Stdin)
	cmd, _ = reader.ReadString('\n')
	cmd = strings.TrimSpace(cmd)

	return cmd
}

func jsonString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	} else if n, ok := v.(json.Number); ok {
		return n.String()
	} else if n, ok := v.(float64); ok {
		return strconv.Itoa(int(n))
	} else {
		fmt.Println("===jsonString unknown type:", reflect.TypeOf(v))
	}
	return ""
}

func startV2ray(verbose bool) (*exec.Cmd, error) {
	v2rayCmd := exec.Command("./sk", "run", "config.json")
	if verbose {
		v2rayCmd.Stdout = os.Stdout
		v2rayCmd.Stderr = os.Stderr
	}

	err := v2rayCmd.Start()
	if err != nil {
		fmt.Println("failed to start v2ray", err)
		return nil, err
	}

	if v2rayCmd.Process != nil {
		fmt.Println("v2ray was started. Pid =", v2rayCmd.Process.Pid)
	}

	return v2rayCmd, nil
}

func stopV2ray(v2rayCmd *exec.Cmd) {
	if v2rayCmd != nil && v2rayCmd.Process != nil {
		if v2rayCmd.ProcessState == nil || !v2rayCmd.ProcessState.Exited() {
			fmt.Println("v2ray will be stopped. Pid =", v2rayCmd.Process.Pid)
			v2rayCmd.Process.Kill()
			v2rayCmd.Wait()
		}
	}

	// double kill
	// cmd := exec.Command("killall", "sk.exe")
	// cmd.Run()
}

func winEnableProxy(enable bool) {
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

func main() {
	var v2rayCmd *exec.Cmd

	var verbose bool
	flag.BoolVar(&verbose, "v", true, "ouput logs of sk.exe")
	flag.Parse()

	tmpl := template.Must(template.New("outbound").Parse(configTemplate))
	tmpl.Option("missingkey=zero") // map

	tmpl2 := template.Must(template.ParseFiles("config.template"))
	tmpl2.Option("missingkey=zero") // map

	// chacha20-ietf-poly1305:G!yBwPWH3Vao@196.247.59.156:804>%ü ?ë
	re := regexp.MustCompile(`(.+):(.+)@(.+):(\d+)`)

	// 104.19.3.16	443	9e6ceeff-2546-3690-ac00-6fcdf31dec94	ws	/chcar	tls
	// ingress-i1.onebox6.org	38701	79386685-16da-327c-9e14-aa6d702d86bc	ws	/hls/cctv5phd.m3u8
	re2 := regexp.MustCompile(`\s*(\S+)\s+(\d+)\s+([\w-]+)\s+(ws|tcp)\s+(\S+)\s*(tls)*`)

	// ss://YWVzLTEyOC1jZmI6UWF6RWRjVGdiMTU5QCQq@14.29.124.174:11050#%F0%9F%87%AD
	re3 := regexp.MustCompile(`(.+)@(.+):(\d+)`)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP,
		syscall.SIGQUIT)

	go func() {
		for s := range signalChan {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				winEnableProxy(false)
				os.Exit(0)
			}
		}
	}()

	for {
		url := readUrl()
		var config map[string]string = make(map[string]string)

		if url == "e" {
			winEnableProxy(true)
			stopV2ray(v2rayCmd)
			v2rayCmd, _ = startV2ray(verbose)
			fmt.Println("System proxy is enabled")
			continue
		} else if url == "d" {
			winEnableProxy(false)
			stopV2ray(v2rayCmd)
			v2rayCmd = nil
			fmt.Println("System proxy is disabled")
			continue
		}

		if strings.HasPrefix(url, "vmess://") {
			url = strings.TrimPrefix(url, "vmess://")
			config["protocol"] = "vmess"

			b, err := base64.StdEncoding.DecodeString(url)
			if err != nil {
				fmt.Println("base64 decode error", err, url)
				continue
			}

			fmt.Println("===vmess json===", string(b))

			var urlJson map[string]any
			err = json.Unmarshal(b, &urlJson)
			if err != nil {
				fmt.Println("json decode error", err, string(b))
				continue
			}

			for k, a := range urlJson {
				v := jsonString(a)
				switch k {
				case "add":
					config["address"] = v
				case "port":
					config["port"] = v
				case "id":
					config["id"] = v
				case "aid":
					config["alterId"] = v
				case "net":
					config["network"] = v
				case "tls":
					if len(v) > 0 {
						config["security"] = "tls"
					}
				case "sni":
					if len(v) > 0 {
						config["sni"] = v
					}
				case "path":
					config["path"] = v
				case "host":
					if len(v) > 0 {
						config["host"] = v
					}
				default:
					config[k] = v
				}
			}
		} else if strings.HasPrefix(url, "vless://") || strings.HasPrefix(url, "trojan://") {
			u, err := neturl.Parse(url)
			if err != nil {
				fmt.Println("parse vless url error", err, url)
				continue
			}
			config["protocol"] = u.Scheme
			config["ps"] = u.Fragment
			config["address"] = u.Hostname()
			config["port"] = u.Port()
			if u.Scheme == "vless" {
				config["id"] = u.User.Username()
			} else {
				config["password"] = u.User.Username() // trojan://
			}
			q := u.Query()

			for k, v := range q {
				switch k {
				case "encryption":
					config["encryption"] = v[0]
				case "sni":
					config["sni"] = v[0]
				case "flow":
					config["flow"] = v[0]
				case "type":
					config["network"] = v[0]
				case "security":
					config["security"] = v[0]
				case "path":
					config["path"] = v[0]
				case "host":
					config["host"] = v[0]
				default:
					config[k] = v[0]
				}
			}
		} else if strings.HasPrefix(url, "ss://") {
			url = strings.TrimPrefix(url, "ss://")
			config["protocol"] = "shadowsocks"

			before, after, found := strings.Cut(url, "#")
			if found {
				url = before
				comments, err := neturl.QueryUnescape(after)
				if err != nil {
					comments = after
				}
				config["ps"] = comments
			}

			re3Fields := re3.FindStringSubmatch(url)
			if re3Fields != nil && len(re3Fields) == 4 {
				b, err := base64.StdEncoding.DecodeString(re3Fields[1])
				if err != nil {
					fmt.Println("base64 decode error", err, re3Fields[1])
					continue
				}
				url = string(b)
				method, pwd, pwdFound := strings.Cut(url, ":")
				if !pwdFound {
					fmt.Println("invalid method and pwd", url)
					continue
				}

				if strings.HasSuffix(method, "cfb") {
					fmt.Println(method, "is not supported")
					continue
				}

				config["method"] = method
				config["password"] = pwd
				config["address"] = re3Fields[2]
				config["port"] = re3Fields[3]
			} else {
				b, err := base64.StdEncoding.DecodeString(url)
				if err != nil {
					fmt.Println("base64 decode error", err, url)
					continue
				}
				url = string(b)
				fmt.Println("===ss url===", url)

				fields := re.FindStringSubmatch(url)
				if fields == nil {
					fmt.Println("invalid ss url", url)
					continue
				}

				if strings.HasSuffix(fields[1], "cfb") {
					fmt.Println(fields[1], "is not supported")
					continue
				}

				config["method"] = fields[1]
				config["password"] = fields[2]
				config["address"] = fields[3]
				config["port"] = fields[4]
			}

		} else {
			fields := re2.FindStringSubmatch(url)
			if fields == nil || (len(fields) != 6 && len(fields) != 7) {
				fmt.Println("unknown url format", url)
				continue
			}

			config["protocol"] = "vmess"
			config["address"] = fields[1]
			config["port"] = fields[2]
			config["id"] = fields[3]
			config["network"] = fields[4]
			config["path"] = fields[5]
			if len(fields) == 7 && fields[6] == "tls" {
				config["security"] = "tls"
			}
		}

		err := tmpl.Execute(os.Stdout, config)
		if err != nil {
			fmt.Printf("template execute error %+v  %+v\n", config, err)
			continue
		}

		configFile, fileErr := os.Create("config.json")
		if fileErr != nil {
			fmt.Println("create config.json error", err)
			continue
		}
		defer configFile.Close()

		err = tmpl2.Execute(configFile, config)
		if err != nil {
			fmt.Printf("template execute error %+v  %+v\n", config, err)
		}

		stopV2ray(v2rayCmd)
		v2rayCmd, _ = startV2ray(verbose)
	}
}
