package health

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

func handler(resp http.ResponseWriter, req *http.Request) {
	consul, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		http.Error(resp, "[ERR] Unable to create a consul client for health checks", 500)
	}
	queryOpt := &consulapi.QueryOptions{}
	lastBackup, _, err := consul.KV().Get("service/consul-snapshot/lastbackup", queryOpt)
	if err != nil || lastBackup == nil {
		http.Error(resp, "No previous backup detected or unable to get backup key!", 500)
	}

	lastTimestamp := string(lastBackup.Value)

	timestampInt, err := strconv.ParseInt(lastTimestamp, 10, 64)
	if err != nil {
		http.Error(resp, "[ERR] Unable to convert last timestamp to int", 500)
	}

	nowtime := time.Now().Unix()

	timediff := nowtime - timestampInt

	if timediff > 3600 {
		http.Error(resp, "[ERR] Backup older than 1 hour", 500)
	} else {
		msg := fmt.Sprintf("Last backup %v seconds ago", timediff)
		resp.Write([]byte(msg))
	}

}

func StartServer() {
	http.HandleFunc("/health", handler)
	http.ListenAndServe(":5001", nil)
}
