package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
)

func main() {
	uuid := os.Getenv("UUID")
	if uuid == "" {
		log.Fatal("UUID env required")
	}
	wsPath := os.Getenv("WS_PATH")
	if wsPath == "" {
		wsPath = "/ray"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	cfgJSON := fmt.Sprintf(`{
	  "log": {"loglevel": "warning"},
	  "inbounds": [{
	    "listen": "127.0.0.1",
	    "port": 10001,
	    "protocol": "vless",
	    "settings": {"clients": [{"id": "%s"}], "decryption": "none"},
	    "streamSettings": {"network": "ws", "wsSettings": {"path": "%s"}}
	  }],
	  "outbounds": [{"protocol": "freedom", "tag": "direct"}]
	}`, uuid, wsPath)
	cfg, err := serial.LoadJSONConfig(strings.NewReader(cfgJSON))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	instance, err := core.New(cfg)
	if err != nil {
		log.Fatalf("create instance: %v", err)
	}
	if err := instance.Start(); err != nil {
		log.Fatalf("start instance: %v", err)
	}
	defer instance.Close()

	target, _ := url.Parse("http://127.0.0.1:10001")
	proxy := httputil.NewSingleHostReverseProxy(target)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == wsPath && strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			proxy.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	log.Printf("listening on :%s, ws path %s", port, wsPath)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
