package proxmox

// https://pve.proxmox.com/wiki/Proxmox_VE_API#API_Tokens
// https://pve.proxmox.com/pve-docs/api-viewer/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"golang.org/x/exp/rand"
)

func proxmoxRequest(method string, path string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%v%v", os.Getenv("PVE_HOST"), path), bytes.NewReader(body))
	if err != nil {
		fmt.Println("ERROR: %v", err.Error())
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("PVEAPIToken=%v!%v=%v", os.Getenv("PVE_USER"), os.Getenv("PVE_TOKENID"), os.Getenv("PVE_UUID")))
	return req, nil
}


type PVEVMs struct {
	Data []struct {
		Node      string  `json:"node"`
		Diskwrite int     `json:"diskwrite"`
		Status    string  `json:"status"`
		Maxmem    int     `json:"maxmem"`
		Uptime    int     `json:"uptime"`
		Mem       int     `json:"mem"`
		Netout    int     `json:"netout"`
		Diskread  int     `json:"diskread"`
		Maxcpu    int     `json:"maxcpu"`
		Pool      string  `json:"pool"`
		Netin     int     `json:"netin"`
		Cpu       float64 `json:"cpu"`
		Template  int     `json:"template"`
		Type      string  `json:"type"`
		Vmid      int     `json:"vmid"`
		Maxdisk   int     `json:"maxdisk"`
		Disk      int     `json:"disk"`
		Id        string  `json:"id"`
		Name      string  `json:"name"`
	} `json:"data"`
}


func GetAllVMs() (*PVEVMs, error) {
	req, err := proxmoxRequest(http.MethodGet, "/api2/json/cluster/resources", []byte{})
	if err != nil {
		fmt.Println("ERROR: %v", err.Error())
		return nil, err
	}
	q := req.URL.Query()
	q.Set("type", "vm")
	req.URL.RawQuery = q.Encode()

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("ERROR: %v", err.Error())
		return nil, err
	}

	var vms PVEVMs
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("ERROR: %v", err.Error())
		return nil, err
	}
	err = json.Unmarshal(body, &vms)
	if err != nil {
		fmt.Println("ERROR: %v", err.Error())
		return nil, err
	}

	return &vms, nil
}


func IsHostnameTaken(hostname string) (bool, error) {
	// Check if hostname is already taken
	vms, err := GetAllVMs()
	if err != nil {
		return false, err
	}

	for _, m := range vms.Data {
		if m.Name == hostname {
			return true, nil
		}
	}

	return false, nil
}
