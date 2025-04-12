package proxmox

// https://pve.proxmox.com/wiki/Proxmox_VE_API#API_Tokens
// https://pve.proxmox.com/pve-docs/api-viewer/

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"github.com/google/uuid"
	"github.com/melbahja/goph"
	"github.com/pkg/sftp"
)

func proxmoxMakeRequest(method string, path string, body []byte) (*http.Request, *http.Client, error) {
	url, err := url.Parse(fmt.Sprintf("%v%v", config.AppConfig.PVE_HOST, path))
	if err != nil {
		return nil, nil, fmt.Errorf("Making request: Parsing URL: %v", err.Error())
	}
	// log.Println("Requesting URL: '" + url.String() + "'")
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("Making request %v %v: %v", method, url, err.Error())
	}

	req.Header.Add("Authorization", fmt.Sprintf("PVEAPIToken=%v!%v=%v", config.AppConfig.PVE_USER, config.AppConfig.PVE_TOKENID, config.AppConfig.PVE_UUID))

	return req, &http.Client{Timeout: time.Second * 10}, nil
}

func proxmoxDoRequest(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Doing request %v %v: %v", req.Method, req.URL, err.Error())
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Doing request: Cannot read body: %v", err.Error())
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Doing request: Status %v\nBody: %v", res.Status, string(body))
	}

	return body, nil
}

// GET /api2/json/nodes
func GetAllClusterNodes() (*[]PVENode, error) {
	req, client, err := proxmoxMakeRequest(http.MethodGet, "/api2/json/nodes", []byte{})
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox nodes: %v\n", err.Error())
	}
	q := req.URL.Query()
	// q.Set("type", "node")
	req.URL.RawQuery = q.Encode()

	body, err := proxmoxDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox nodes: %v", err.Error())
	}

	var nodes pveNodeList
	err = json.Unmarshal(body, &nodes)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox nodes: Unmarshal error: %v", err.Error())
	}

	return &nodes.Data, nil
}

type PVENode struct {
	Status  string  `json:"status"`
	Disk    int     `json:"disk"`
	Maxdisk int     `json:"maxdisk"`
	Mem     int     `json:"mem"`
	Maxmem  int     `json:"maxmem"`
	Cpu     float32 `json:"cpu"`
	Type    string  `json:"type"`
	Id      string  `json:"id"`
	Node    string  `json:"node"`
}
type pveNodeList struct {
	Data []PVENode `json:"data"`
}

// GET /api2/json/nodes/{node}/qemu
func GetAllNodeVMs(node string) (*[]PVENodeVM, error) {
	req, client, err := proxmoxMakeRequest(http.MethodGet, fmt.Sprintf("/api2/json/nodes/%v/qemu", node), []byte{})
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: %v", err.Error())
	}
	q := req.URL.Query()
	q.Set("full", "1")
	req.URL.RawQuery = q.Encode()

	body, err := proxmoxDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: %v", err.Error())
	}

	var vms pveNodeVMList
	err = json.Unmarshal(body, &vms)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: Unmarshal error: %v", err.Error())
	}

	return &(vms.Data), nil
}

func GetAllNodeVMsByName(node string, name string) (*[]PVENodeVM, error) {
	nodes, err := GetAllNodeVMs(node)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve VMs by name on node %v: %v", node, err.Error())
	}

	var vms []PVENodeVM
	for _, vm := range *nodes {
		if vm.Name == name {
			vms = append(vms, vm)
		}
	}

	return &vms, nil
}

// GET /api2/json/nodes/{node}/qemu/{vmid}/status/current
func GetNodeVM(node string, vm_id int) (*PVENodeVM, error) {
	req, client, err := proxmoxMakeRequest(http.MethodGet, fmt.Sprintf("/api2/json/nodes/%v/qemu/%v/status/current", node, vm_id), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve VM '%v' on node '%v': %v", vm_id, node, err.Error())
	}

	body, err := proxmoxDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve VM '%v' on node '%v': %v", vm_id, node, err.Error())
	}

	var vm pveNodeVM
	err = json.Unmarshal(body, &vm)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve VM '%v' on node '%v': Unmarshal error: %v", vm_id, node, err.Error())
	}

	return &vm.Data, nil
}

type PVENodeVM struct {
	Status string `json:"status"`
	Vmid   int    `json:"vmid"`

	Agent           int     `json:"agent"`
	Clipboard       string  `json:"clipboard"`
	Cpus            float64 `json:"cpus"`
	Diskread        int     `json:"diskread"`
	Diskwrite       int     `json:"diskwrite"`
	Lock            string  `json:"lock"`
	Maxdisk         int     `json:"maxdisk"`
	Maxmem          int     `json:"maxmem"`
	Name            string  `json:"name"`
	Netin           int     `json:"netin"`
	Netout          int     `json:"netout"`
	Pid             int     `json:"pid"`
	Qmpstatus       string  `json:"qmpstatus"`
	Running_machine string  `json:"running-machine"`
	Running_qemu    string  `json:"running-qemu"`
	Tags            string  `json:"tags"`
	Template        bool    `json:"template"`
	Uptime          int     `json:"uptime"`

	Spice bool `json:"spice"`
}
type pveNodeVMList struct {
	Data []PVENodeVM `json:"data"`
}
type pveNodeVM struct {
	Data PVENodeVM `json:"data"`
}

// GET /api2/json/cluster/resources?type=vm
func GetAllClusterVMs() (*[]PVEClusterVM, error) {
	req, client, err := proxmoxMakeRequest(http.MethodGet, "/api2/json/cluster/resources", []byte{})
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: %v", err.Error())
	}
	q := req.URL.Query()
	q.Set("type", "vm")
	req.URL.RawQuery = q.Encode()

	body, err := proxmoxDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: %v", err.Error())
	}

	var vms pveClusterVMList
	err = json.Unmarshal(body, &vms)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: Unmarshal error: %v", err.Error())
	}

	return &(vms.Data), nil
}

type PVEClusterVM struct {
	Id   string `json:"id"`
	Type string `json:"type"`

	Cgroup_mode int     `json:"cgroup-mode"`
	Cpu         float64 `json:"cpu"`
	Disk        int     `json:"disk"`
	Diskread    int     `json:"diskread"`
	Diskwrite   int     `json:"diskwrite"`
	Hastate     string  `json:"hastate"`
	Level       string  `json:"level"`
	Lock        string  `json:"lock"`
	Maxcpu      float64 `json:"maxcpu"`
	Maxdisk     int     `json:"maxdisk"`
	Maxmem      int     `json:"maxmem"`
	Mem         int     `json:"mem"`
	Name        string  `json:"name"`
	Netin       int     `json:"netin"`
	Netout      int     `json:"netout"`
	Node        string  `json:"node"`
	Plugintype  string  `json:"plugintype"`
	Pool        string  `json:"pool"`
	Status      string  `json:"status"`
	Storage     string  `json:"storage"`
	Tags        string  `json:"tags"`
	Template    int     `json:"template"`
	Uptime      int     `json:"uptime"`
	Vmid        int     `json:"vmid"`
}
type pveClusterVMList struct {
	Data []PVEClusterVM `json:"data"`
}

// POST /api2/json/nodes/{node}/qemu/{vmid}/status/start
func ForceStopNodeVM(node string, vm_id int) error {
	req, client, err := proxmoxMakeRequest(http.MethodPost, fmt.Sprintf("/api2/json/nodes/%v/qemu/%v/status/stop", node, vm_id), nil)
	if err != nil {
		return fmt.Errorf("Failed to force stop VM '%v' on node '%v': %v", vm_id, node, err.Error())
	}

	_, err = proxmoxDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Failed to force stop VM '%v' on node '%v': %v", vm_id, node, err.Error())
	}

	// Wait for VM to stop
	for {
		vm, err := GetNodeVM(node, vm_id)
		if err != nil {
			return fmt.Errorf("Failed to force stop VM '%v' on node '%v': Waiting for VM to stop: %v", vm_id, node, err.Error())
		}
		if vm.Status == "stopped" {
			break
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("[+] Stopped VM %v on node %v\n", vm_id, node)
	return nil
}

// DELETE /api2/json/nodes/{node}/qemu/{vmid}
func DeleteNodeVM(node string, vm_id int, destroy_unreferenced_disks bool, purge_vm_from_configs bool, skip_lock bool) error {
	req, client, err := proxmoxMakeRequest(http.MethodDelete, fmt.Sprintf("/api2/json/nodes/%v/qemu/%v", node, vm_id), nil)
	if err != nil {
		return fmt.Errorf("Failed to delete VM '%v' on node '%v': %v", vm_id, node, err.Error())
	}
	q := req.URL.Query()
	q.Set("destroy-unreferenced-disks", map[bool]string{true: "1", false: "0"}[destroy_unreferenced_disks])
	q.Set("purge", map[bool]string{true: "1", false: "0"}[purge_vm_from_configs])
	// Skips locks (usually the VM running). Only root can use.
	q.Set("skiplock", map[bool]string{true: "1", false: "0"}[skip_lock])
	req.URL.RawQuery = q.Encode()

	_, err = proxmoxDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Failed to delete VM '%v' on node '%v': %v", vm_id, node, err.Error())
	}

	log.Printf("[+] Deleted VM %v on node %v [destroy-unreferenced-disks: %v, purge: %v, skiplock: %v]\n", vm_id, node, destroy_unreferenced_disks, purge_vm_from_configs, skip_lock)
	return nil
}

type VMCreationOptions struct {
	Template     string
	FQDN         string
	Reinstall    bool
	Cores_CPU    int
	RAM_MB       int64
	Disk_GB      int64
	UseQemuAgent bool
	Tags         []string
	Notes        string
	SSHPubkeys   []string
}

const (
	IMAGE_UBUNTU_22_04 = "Ubuntu 22.04 - Jammy Jellyfish"
	IMAGE_UBUNTU_24_04 = "Ubuntu 24.04 - Noble Numbat"
	IMAGE_DEBIAN_12    = "Debian 12 - Bookworm"
	IMAGE_DEBIAN_11    = "Debian 11 - Bullseye"
)

func CreateVM(options VMCreationOptions) (*PVENodeVM, error) {

	//! Verify that configured CM SSH host is actually a cluster management node
	// fmt.Println("[-] Checking if running on a cluster management node")
	client, err := createCMSSHClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: %v", err)
	}
	defer client.Close()

	log.Println("[-] Checking if CM SSH session is actually on a cluster management node")
	stdout, err := client.Run("hostname --fqdn")
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Cannot verify hostname of configured CM host: %v\nOutput:\n%s", err, stdout)
	}
	strstdout := strings.Trim(string(stdout), " \n")
	match, _ := regexp.MatchString("^cm-.+\\.sos\\.ethz\\.ch$", strstdout)
	if !match {
		return nil, fmt.Errorf("Failed to create VM: Configured CM SSH host is not a cluster management node")
	}

	//! Prepare default/hardcoded parameters

	comp_node := config.AppConfig.SSH_COMP_HOST
	comp_node_name := config.AppConfig.COMP_NAME
	example_fqdn := "example.vsos.ethz.ch"
	net := "vm"
	CEPH_POOL := "ssd"
	VM_NETMASK_4 := 24
	VM_GATEWAY_4 := "192.33.91.1"
	VM_NETMASK_6 := 118
	VM_GATEWAY_6 := "2001:67c:10ec:49c3::1"
	TEMPLATE_STORAGE := "/srv/cnfs"
	TEMPLATE_STORAGE_ON_COMP := "/mnt/pve/cnfs"
	VM_SWAP_SIZE := "512M"
	VM_DEFAULT_ROOT_SIZE := "15G"
	VM_DEFAULT_RAM_SIZE := "2048"

	// options.Template := options.Template
	ssh_user := "root"
	first_boot_line := "no"

	VMPUBKEY_PATH := "/root/.ssh/vm_univ_pubkey.key"

	//! Choosing appropriate user and first boot line
	log.Println("[-] Choosing appropriate user and first boot line based on template")
	switch options.Template {
	case IMAGE_DEBIAN_11:
		options.Template = "bullseye"
		ssh_user = "debian"
		first_boot_line = "Cloud-init .* finished"
	case IMAGE_DEBIAN_12:
		options.Template = "bookworm"
		ssh_user = "debian"
		first_boot_line = "Cloud-init .* finished"
	case IMAGE_UBUNTU_22_04:
		options.Template = "jammy"
		ssh_user = "ubuntu"
		first_boot_line = "Cloud-init .* finished"
	case IMAGE_UBUNTU_24_04:
		options.Template = "noble"
		ssh_user = "ubuntu"
		first_boot_line = "Cloud-init .* finished"
	default:
		return nil, fmt.Errorf("\tFailed to create VM: Unknown template %v", options.Template)
	}

	//! Checking existence of DNS entries for chosen FQDN
	log.Printf("[-] Checking existence of DNS entries for chosen FQDN %v", options.FQDN)
	ipv4s, ipv6s, err := netcenter.GetHostIPs(options.FQDN)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to check if there are existing ipv4 DNS entries for FQDN: %v", err.Error())
	}

	// Map ipv4s object to just their ips
	var ipv4s_str []string
	for _, ip := range ipv4s {
		ipv4s_str = append(ipv4s_str, ip.IP.String())
	}
	log.Println("\tIPv4: ", strings.Join(ipv4s_str, ", "))

	// Map ipv6s object to just their ips
	var ipv6s_str []string
	for _, ip := range ipv6s {
		ipv6s_str = append(ipv6s_str, ip.IP.String())
	}
	log.Println("\tIPv6: ", strings.Join(ipv6s_str, ", "))

	// TODO: What is actually allowed ?
	if len(ipv4s_str) > 1 || len(ipv6s_str) > 1 {
		return nil, fmt.Errorf("Failed to create VM: Each VM hostname %v should have AT MOST one IPv4 and one IPv6 address.", options.FQDN)
	}

	if options.Reinstall && (len(ipv4s_str) == 0 || len(ipv6s_str) == 0) {
		return nil, fmt.Errorf("Failed to create VM: Cannot reinstall VM with FQDN %v as it does not have both ipv4 and ipv6 DNS entries", options.FQDN)
	}

	if !options.Reinstall && (len(ipv4s_str) > 0 || len(ipv6s_str) > 0) {
		log.Println("\t[!] FQDN still has DNS entries with IP addresses:")
		// TODO: You sure you want to continue?
		return nil, nil
	}

	//! Check if the image exists on the management node
	IMAGE := fmt.Sprintf("%v/cloudinit/current-%v-amd64.qcow2", TEMPLATE_STORAGE, options.Template)
	IMAGE_REMOTE := fmt.Sprintf("%v/cloudinit/current-%v-amd64.qcow2", TEMPLATE_STORAGE_ON_COMP, options.Template)

	log.Printf("[-] Checking if image '%v' exists on management node", IMAGE)

	cm_sftp, err := createCMSFTPClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: CM SFTP: %v", err.Error())
	}
	defer cm_sftp.Close()

	_, err = cm_sftp.Stat(IMAGE)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Cannot ensure existence of '%v' on CM node: %v", IMAGE, err)
	}

	//! Preparing sources.list for VM
	log.Println("[-] Preparing apt sources for VM")
	var SOURCES_LIST string
	if options.Template == "bullseye" || options.Template == "bookworm" {
		SOURCES_LIST = fmt.Sprintf(`
		deb http://ftp.ch.debian.org/debian %v main
		#deb-src http://ftp.ch.debian.org/debian %v main

		deb http://ftp.ch.debian.org/debian %v-updates main
		#deb-src http://ftp.ch.debian.org/debian %v-updates main

		deb http://security.debian.org/ %v-security main
		#deb-src http://security.debian.org/ %v-security main`, options.Template, options.Template, options.Template, options.Template, options.Template, options.Template)
	} else if options.Template == "jammy" || options.Template == "noble" {
		SOURCES_LIST = fmt.Sprintf(`
		deb http://ch.archive.ubuntu.com/ubuntu %v main universe multiverse
		#deb-src http://ch.archive.ubuntu.com/ubuntu %v main universe multiverse

		deb http://ch.archive.ubuntu.com/ubuntu %v-updates main universe multiverse
		#deb-src http://ch.archive.ubuntu.com/ubuntu %v-updates main universe multiverse

		deb http://security.ubuntu.com/ubuntu %v-security main universe multiverse
		#deb-src http://security.ubuntu.com/ubuntu %v-security main universe multiverse`, options.Template, options.Template, options.Template, options.Template, options.Template, options.Template)
	} else {
		return nil, fmt.Errorf("Failed to create VM: Unknown template %v", options.Template)
	}

	//! Generate random VM ID
	// TODO: Do not generate randomly, rather take the smallest available one
	log.Println("[-] Generating random VM ID")
	VM_ID := 100000 + rand.Intn(899999)

	//! Summary
	log.Printf(`
SUMMARY
-------
VM_ID: %v
FQDN: %v
Description: %v

OS: %v
Cores: %v
RAM: %v
Disk size: %v
Swap size: %v
Ceph pool: %v

QEMU agent: %v

Reinstall: %v
-------
`, VM_ID, options.FQDN, options.Notes, options.Template, options.Cores_CPU, options.RAM_MB, options.Disk_GB, VM_SWAP_SIZE, CEPH_POOL, options.UseQemuAgent, options.Reinstall)

	//! Register DNS entries for FQDN and an available IPv4 and IPv6 address.
	if !options.Reinstall {
		log.Printf("[-] Registering \"FQDN\" %v in net \"%v\"\n", options.FQDN, net)
		ipv4, ipv6, err := netcenter.Registerhost(net, options.FQDN)
		if err != nil {
			return nil, fmt.Errorf("Failed to create VM: %v", err)
		}
		ipv4s_str = append(ipv4s_str, (*ipv4).String())
		ipv6s_str = append(ipv6s_str, (*ipv6).String())
	}

	//! Read universal VM public key
	// TODO: Startup check
	log.Println("[-] Reading universal VM public key from file")

	vmpubkey_content, err := os.ReadFile(VMPUBKEY_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to open the universal public VM key '%v': %v", VMPUBKEY_PATH, err)
	}

	//! Prepare authorized_keys file
	log.Println("[-] Preparing authorized_keys file")
	log.Println("\tConcatenating VM universal public key with provided pubkeys")
	authorized_keys_content := strings.Join(slices.Concat(options.SSHPubkeys, strings.Split(string(vmpubkey_content), "\n")), "\n\n")

	//! Upload authorized_keys file to comp node
	VM_AUTHORIZED_KEYS_PATH := fmt.Sprintf("/tmp/vmwiz-%v.ssh.pub", VM_ID)
	log.Printf("[-] Uploading authorized_keys file to %v:%v\n", comp_node, VM_AUTHORIZED_KEYS_PATH)
	comp_sftp, err := createCompSFTPClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: %v", err.Error())
	}

	comp_sftp_authorized_keys, err := comp_sftp.Create(VM_AUTHORIZED_KEYS_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to create file '%v': %v", VM_AUTHORIZED_KEYS_PATH, err)
	}
	defer comp_sftp.Remove(VM_AUTHORIZED_KEYS_PATH)
	defer comp_sftp_authorized_keys.Close()

	_, err = comp_sftp_authorized_keys.Write([]byte(authorized_keys_content))
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to write to file '%v': %v", VM_AUTHORIZED_KEYS_PATH, err)
	}

	//! Prepare Cloudinit configuration
	log.Println("[-] Preparing Cloudinit configuration")
	cloudinit_fragments := fmt.Sprintf("ipconfig0: gw=%s,ip=%s/%d,ip6=%s/%d", VM_GATEWAY_4, ipv4s_str[0], VM_NETMASK_4, ipv6s_str[0], VM_NETMASK_6)
	// fmt.Println(cloudinit_fragments)

	//! Upload Cloudinit configuration to comp node
	VM_CLOUDINIT_PATH := fmt.Sprintf("/tmp/vmwiz-%v.cloudinit.tail", VM_ID)
	log.Printf("[-] Uploading Cloudinit fragments to %v:%v\n", comp_node, VM_CLOUDINIT_PATH)
	comp_sftp_cloudinitfrags, err := comp_sftp.Create(VM_CLOUDINIT_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to create file '%v': %v", VM_CLOUDINIT_PATH, err)
	}
	defer comp_sftp_cloudinitfrags.Close()
	defer comp_sftp.Remove(VM_CLOUDINIT_PATH)

	_, err = comp_sftp_cloudinitfrags.Write([]byte(cloudinit_fragments))
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to write to file '%v': %v", VM_CLOUDINIT_PATH, err)
	}

	//! Create VM on compute node
	log.Printf("[-] Creating VM on %v\n", comp_node)

	VM_NETMODEL := "virtio"

	VM_FQDN := options.FQDN
	IMAGE_PATH := IMAGE_REMOTE
	FIRST_BOOT_LINE := first_boot_line
	VM_DESC := options.Notes
	var AGENT int
	if options.UseQemuAgent {
		AGENT = 1
	} else {
		AGENT = 0
	}
	SWAP_SIZE := VM_SWAP_SIZE

	EFI_DISK_NAME := fmt.Sprintf("vm-%v-efivars", VM_ID)
	SWAP_DISK_NAME := fmt.Sprintf("vm-%v-disk-0", VM_ID)
	MAIN_DISK_NAME := fmt.Sprintf("vm-%v-disk-1", VM_ID)

	//! Verify that configured Comp node SSH host is actually a compute node
	log.Println("\t[-] Checking if compute SSH session is actually on a compute node")
	comp_ssh, err := createCompSSHClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: %v", err)
	}
	defer comp_ssh.Close()

	stdout, err = comp_ssh.Run("hostname --fqdn")
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot verify hostname of configured compute node: %v\nOutput:\n%s", err, stdout)
	}
	strstdout = strings.Trim(string(stdout), " \n")
	match, _ = regexp.MatchString("^comp-.+\\.sos\\.ethz\\.ch$", strstdout)
	if !match {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Configured compute SSH host is not a compute node")
	}

	//! Generate MAC address
	// TODO: Fix collisions by ensuring we are taking a free mac addr
	r := regexp.MustCompile("^(..)(..)(..)(..)(..).*$")

	digest := md5.Sum([]byte(VM_FQDN))
	hex_digest := hex.EncodeToString(digest[:])

	matches := r.FindAllStringSubmatch(hex_digest, 5)[0][1:]

	// Mac addresses have to start with 02 because they are unicast and locally administrated.
	// If it doesnt start with 02, Proxmox might remove the network interface from the VM.
	VM_MACADDR := fmt.Sprintf("02:%s:%s:%s:%s:%s", matches[0], matches[1], matches[2], matches[3], matches[4])

	if len(matches) != 5 {
		return nil, fmt.Errorf("Failed to create VM: Failed to generate MAC address: Generated MAC address is not 12 bytes long")
	}

	log.Printf("\t[-] Generated MAC address: %v\n", VM_MACADDR)

	//! Create disks
	log.Printf("\t[-] Creating disks\n")

	// Create swap disk
	command := fmt.Sprintf("rbd -p \"%v\" create --size \"%v\" \"%v\"", CEPH_POOL, SWAP_SIZE, SWAP_DISK_NAME)
	log.Printf("\t\t[-] Creating SWAP disk\n")
	log.Printf("\t\t> %v", command)
	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot create SWAP disk: %v\nOutput:\n%s", err, stdout)
	}

	// Create EFI disk
	command = fmt.Sprintf("rbd -p \"%v\" create --size \"4M\" \"%v\"", CEPH_POOL, EFI_DISK_NAME)
	log.Printf("\t\t[-] Creating EFI disk\n")
	log.Printf("\t\t> %v\n", command)
	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot create EFI disk: %v\nOutput:\n%s", err, stdout)
	}

	//! Creating VM configuration in Proxmox
	log.Printf("\t[-] Generating VM configuration\n")
	VM_CONF_TEMPLATE_PATH := "proxmox/VM.conf.tmpl"
	uuidv7, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to generate UUID: %v", err)
	}

	vm_config := new(bytes.Buffer)
	vm_config_template, err := template.ParseFiles(VM_CONF_TEMPLATE_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to parse template: %v", err)
	}
	err = vm_config_template.Execute(vm_config, struct {
		AGENT        int
		VM_DESC      string
		VM_FQDN      string
		CEPH_POOL    string
		EFI_DISK     string
		CPU_CORES    int
		RAM_SIZE     int
		SWAP_DISK    string
		SWAP_SIZE    string
		TAGS         string
		UUIDV7       string
		VM_GATEWAY_4 string
		IPV4S_STR0   string
		VM_NETMASK_4 int
		IPV6S_STR0   string
		VM_NETMASK_6 int
	}{
		AGENT:        AGENT,
		VM_DESC:      VM_DESC,
		VM_FQDN:      VM_FQDN,
		CEPH_POOL:    CEPH_POOL,
		EFI_DISK:     EFI_DISK_NAME,
		CPU_CORES:    options.Cores_CPU,
		RAM_SIZE:     int(options.RAM_MB),
		SWAP_DISK:    SWAP_DISK_NAME,
		SWAP_SIZE:    SWAP_SIZE,
		TAGS:         strings.Join(options.Tags, ";"),
		UUIDV7:       uuidv7.String(),
		VM_GATEWAY_4: VM_GATEWAY_4,
		IPV4S_STR0:   ipv4s_str[0],
		VM_NETMASK_4: VM_NETMASK_4,
		IPV6S_STR0:   ipv6s_str[0],
		VM_NETMASK_6: VM_NETMASK_6,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to execute template: %v", err)
	}

	//! Upload VM configuration in Proxmox

	VM_CONFIG_PATH := fmt.Sprintf("/etc/pve/local/qemu-server/%v.conf", VM_ID)

	log.Printf("\t[-] Uploading VM configuration to %v:%v\n", comp_node, VM_CONFIG_PATH)
	vm_config_file, err := comp_sftp.Create(VM_CONFIG_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to create file '%v': %v", VM_CONFIG_PATH, err)
	}
	defer vm_config_file.Close()

	_, err = vm_config_file.Write(vm_config.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to write to file '%v': %v", VM_CONFIG_PATH, err)
	}

	//! Importing disk image
	log.Printf("\t[-] Importing disk image\n")
	command = fmt.Sprintf("qm importdisk \"%v\" \"%v\" \"%v\"", VM_ID, IMAGE_PATH, CEPH_POOL)
	log.Printf("\t\t> %v\n", command)

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot import disk image: %v\nOutput:\n%s", err, stdout)
	}

	//! Attaching disk to VM
	log.Printf("\t[-] Attaching disk to VM\n")
	command = fmt.Sprintf("qm set \"%v\" --scsi0 \"%v:%v,discard=on\"", VM_ID, CEPH_POOL, MAIN_DISK_NAME)
	log.Printf("\t\t> %v \n", command)

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot attach disk to VM: %v\nOutput:\n%s", err, stdout)
	}

	//! Resizing VM root disk to target size
	log.Printf("\t[-] Resizing VM root disk to target size\n")
	command = fmt.Sprintf("qm resize \"%v\" scsi0 \"%v\"", VM_ID, fmt.Sprintf("%vG", options.Disk_GB))
	log.Printf("\t\t> %v \n", command)

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot resize VM root disk: %v\nOutput:\n%s", err, stdout)
	}

	//! Appending cloudinit fragments to VM configuration
	log.Printf("\t[-] Appending cloudinit fragments to VM configuration\n")
	command = fmt.Sprintf("cat \"%v\" >> \"%v\"", VM_CLOUDINIT_PATH, VM_CONFIG_PATH)
	log.Printf("\t\t> %v \n", command)

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot append cloudinit fragments to VM configuration: %v\nOutput:\n%s", err, stdout)
	}

	//! Creating Cloudinit disk
	log.Printf("\t[-] Creating Cloudinit disk\n")
	command = fmt.Sprintf("qm set \"%v\" -scsi2 \"%v:cloudinit\"", VM_ID, CEPH_POOL)
	log.Printf("\t\t> %v \n", command)

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot create Cloudinit disk: %v\nOutput:\n%s", err, stdout)
	}

	//! Adding SSH keys to machine
	log.Printf("\t[-] Adding SSH keys to machine\n")
	command = fmt.Sprintf("qm set \"%v\" --sshkey \"%v\"", VM_ID, VM_AUTHORIZED_KEYS_PATH)
	log.Printf("\t\t> %v \n", command)

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot add SSH keys to machine: %v\nOutput:\n%s", err, stdout)
	}

	//! Append network configuration to VM configuration
	// ? For some reason, running the previous commands erases the network config entry, so we append it here, after running the aforementioned commands
	log.Printf("\t[-] Appending network configuration to VM configuration\n")
	config := fmt.Sprintf("net0: %v=%v,bridge=vmbr1,rate=125", VM_NETMODEL, VM_MACADDR)
	command = fmt.Sprintf("echo \"%v\" >> \"%v\"", config, VM_CONFIG_PATH)
	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot append network configuration to VM configuration: %v\nOutput:\n%s", err, stdout)
	}

	//! Booting VM
	log.Printf("\t[-] Booting VM\n")
	command = fmt.Sprintf("qm start \"%v\"", VM_ID)
	log.Printf("\t\t> %v \n", command)

	vm_boot_start_timestamp := time.Now()

	stdout, err = comp_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Cannot boot VM: %v\nOutput:\n%s", err, stdout)
	}

	//! Wait for VM to be reachable
	log.Println("\t[-] Waiting for VM to complete first setup")
	COMP_VM_BOOT_LOG_PATH := fmt.Sprintf("/tmp/%v.vmwiz.boot.log", VM_ID)
	COMP_QEMU_VM_BOOT_LOG_PATH := fmt.Sprintf("/var/run/qemu-server/%v.serial0", VM_ID)
	comp_boot_log_file, err := comp_sftp.Create(COMP_VM_BOOT_LOG_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to create file '%v': %v", COMP_VM_BOOT_LOG_PATH, err)
	}
	defer comp_sftp.Remove(COMP_VM_BOOT_LOG_PATH)
	defer comp_boot_log_file.Close()

	// Tailing VM's QEMU log file on Comp node
	log.Printf("\t\t[-] Waiting for boot to complete by tailing QEMU's boot log on Comp node at '%v'\n", COMP_QEMU_VM_BOOT_LOG_PATH)
	session, err := comp_ssh.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Failed to create session: %v", err)
	}
	defer session.Close()

	stdout_reader, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Failed to create stdout pipe: %v", err)
	}

	command = fmt.Sprintf("socat -u \"%v\" -", COMP_QEMU_VM_BOOT_LOG_PATH)
	log.Printf("\t\t> %v\n", command)
	if err := session.Start(command); err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SSH: Failed to start tailing qemu log: %v", err)
	}

	// Read output line by line
	scanner := bufio.NewScanner(stdout_reader)
	lines_read := 0
	last_line_timestamp := time.Now()
	for scanner.Scan() {
		lines_read++
		line := scanner.Text()
		if time.Now().Sub(last_line_timestamp) >= 30*time.Second {
			log.Printf("\t\t VM still booting. Elapsed: %v seconds", int(time.Now().Sub(vm_boot_start_timestamp).Seconds()))
			last_line_timestamp = time.Now()
		}
		// Append to file
		_, err := comp_boot_log_file.Write([]byte(line + "\n"))
		if err != nil {
			return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to append to file '%v': %v", COMP_VM_BOOT_LOG_PATH, err)
		}
		match, _ := regexp.MatchString(first_boot_line, line)
		if match {
			break
		}
	}
	log.Println("\t\t [X] VM has completed first boot in ", int(time.Now().Sub(vm_boot_start_timestamp).Seconds()), " seconds")

	// Check for any scanning error
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading output: %v", err)
	}

	//! Copying VM's boot log file from Comp to CM
	log.Printf("[-] Copying VM's boot log file from Comp node to CM at '%v'\n", COMP_VM_BOOT_LOG_PATH)
	log.Println("\t[-] Reading VM's boot log file from Comp node")
	comp_boot_log_file.Close()
	comp_boot_log_file, err = comp_sftp.Open(COMP_VM_BOOT_LOG_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to open file '%v': %v", COMP_VM_BOOT_LOG_PATH, err)
	}
	defer comp_sftp.Remove(COMP_VM_BOOT_LOG_PATH)
	defer comp_boot_log_file.Close()
	comp_sftp_bootlog_content, err := io.ReadAll(comp_boot_log_file)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to read file '%v': %v", COMP_VM_BOOT_LOG_PATH, err)
	}

	CM_VM_BOOT_LOG_PATH_CM := fmt.Sprintf("/tmp/%v.vmwiz.boot.log", VM_ID)
	log.Printf("\t[-] Writing VM's boot log file to CM at '%v'\n", CM_VM_BOOT_LOG_PATH_CM)
	cm_sftp_bootlog_cm, err := cm_sftp.Create(CM_VM_BOOT_LOG_PATH_CM)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: CM node SFTP: Failed to create file '%v': %v", CM_VM_BOOT_LOG_PATH_CM, err)
	}
	defer cm_sftp.Remove(CM_VM_BOOT_LOG_PATH_CM)
	defer cm_sftp_bootlog_cm.Close()
	_, err = cm_sftp_bootlog_cm.Write(comp_sftp_bootlog_content)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: CM node SFTP: Failed to write to file '%v': %v", CM_VM_BOOT_LOG_PATH_CM, err)
	}
	// fmt.Println(string(comp_sftp_bootlog_content))

	//! Adding VM ssh public key to CM known hosts file
	log.Printf("\t[-] Generating VM SSH fingerprints\n")
	cm_ssh, err := createCMSSHClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Generating VM SSH fingerprints: Failed to create CM SSH client: %v", err)
	}
	// Removing previous entries
	command = fmt.Sprintf("ssh-keygen -f \"/root/.ssh/known_hosts\" -R \"%v\"", ipv4s_str[0])
	_, _ = cm_ssh.Run(command)
	command = fmt.Sprintf("ssh-keygen -f \"/root/.ssh/known_hosts\" -R \"%v\"", ipv6s_str[0])
	_, _ = cm_ssh.Run(command)

	log.Printf("\t\t[-] Extracting VM pubkeys from boot log\n")
	vm_pubkeys_regex, err := regexp.Compile("-----BEGIN SSH HOST KEY KEYS-----([\\s\\S]*)-----END SSH HOST KEY KEYS-----")
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Generating VM SSH fingerprints: Failed to compile regex: %v", err)
	}

	vm_pubkey_regex_matches := vm_pubkeys_regex.FindAllStringSubmatch(string(comp_sftp_bootlog_content), -1)
	if vm_pubkey_regex_matches == nil || len(vm_pubkey_regex_matches[0]) == 0 {
		return nil, fmt.Errorf("Failed to create VM: Generating VM SSH fingerprints: Failed to find pubkeys in boot log")
	}

	vm_pubkeys := strings.Split(vm_pubkey_regex_matches[0][1], "\n")

	// Choosing the ed25519 key
	var ssh_ed_25519_pubkey string
	for _, pubkey := range vm_pubkeys {
		if strings.Contains(pubkey, "ssh-ed25519") {
			ssh_ed_25519_pubkey = pubkey
			break
		}
	}
	if ssh_ed_25519_pubkey == "" {
		return nil, fmt.Errorf("Failed to create VM: Generating VM SSH fingerprints: Failed to find ed25519 pubkey in boot log. Found Pubkeys are: %v", vm_pubkeys)
	}

	ssh_ed_25519_pubkey_parts := strings.Fields(ssh_ed_25519_pubkey)
	ssh_ed_25519_pubkey = strings.Join(ssh_ed_25519_pubkey_parts[:2], " ")

	// Append to CM known hosts file
	log.Printf("\t\t[-] Appending VM SSH ed25519 pubkey to CM's known hosts\n")
	command = "echo \"" + options.FQDN + "," + ipv4s_str[0] + "," + ipv6s_str[0] + " " + ssh_ed_25519_pubkey + "\" >> /root/.ssh/known_hosts"
	log.Printf("\t\t> %v\n", command)
	_, err = cm_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Generating VM SSH fingerprints: Failed to append fingerprints to known hosts: %v", err)
	}

	var vm_fingerprints []string
	for _, pubkey := range vm_pubkeys {
		if pubkey == "" {
			continue
		}
		for _, hash_algo := range []string{"sha256", "md5"} {
			command = fmt.Sprintf("echo '%v' | ssh-keygen -f - -l -E %v | awk '{print $2}'", pubkey, hash_algo)
			log.Printf("\t\t> %v\n", command)
			stdout, err = cm_ssh.Run(command)
			if err != nil {
				return nil, fmt.Errorf("Failed to create VM: Generating VM SSH fingerprints: Failed to get fingerprint: %v", err)
			}

			pubkey_type := strings.Fields(pubkey)[0]
			vm_fingerprints = append(vm_fingerprints, fmt.Sprintf("%v %v", pubkey_type, strings.Trim(string(stdout), " \n")))
		}
	}

	//! Prepare VM post-install script
	POST_INSTALL_SCRIPT_TEMPLATE_PATH := "proxmox/vm_finish_script.sh.tmpl"
	log.Printf("\t[-] Preparing VM post-install script from template '%v'\n", POST_INSTALL_SCRIPT_TEMPLATE_PATH)
	vm_finish_script_content := new(bytes.Buffer)
	post_install_template, err := template.ParseFiles(POST_INSTALL_SCRIPT_TEMPLATE_PATH)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to parse template: %v", err)
	}
	err = post_install_template.Execute(vm_finish_script_content, struct {
		SOURCES_LIST string
		VM_GATEWAY_6 string
		UseQemuAgent bool
	}{
		SOURCES_LIST: SOURCES_LIST,
		VM_GATEWAY_6: VM_GATEWAY_6,
		UseQemuAgent: options.UseQemuAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to execute template: %v", err)
	}

	//! Upload post-install script to VM
	log.Printf("\t[-] Uploading post-install script to VM\n")
	POST_INSTALL_SCRIPT_PATH_CM := fmt.Sprintf("/tmp/%v-vmwiz-post-install.sh", VM_ID)
	POST_INSTALL_SCRIPT_PATH_VM := fmt.Sprintf("/home/%v/vmwiz-post-install.sh", ssh_user)
	POST_INSTALL_LOG_PATH_CM := fmt.Sprintf("/tmp/%v-vmwiz-post-install.log", VM_ID)
	log.Printf("\t\t[-] Creating post-install script to CM first at %v\n", POST_INSTALL_SCRIPT_PATH_CM)
	cm_sftp_postinstall, err := cm_sftp.Create(POST_INSTALL_SCRIPT_PATH_CM)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: CM SFTP: Failed to create file '%v': %v", POST_INSTALL_SCRIPT_PATH_CM, err)
	}
	defer cm_sftp.Remove(POST_INSTALL_SCRIPT_PATH_CM)
	defer cm_sftp_postinstall.Close()

	_, err = cm_sftp_postinstall.Write(vm_finish_script_content.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: CM SFTP: Failed to write to file '%v': %v", POST_INSTALL_SCRIPT_PATH_CM, err)
	}

	log.Printf("\t\t[-] Copying post-install script from CM to VM\n")
	command = fmt.Sprintf("scp %v %v@%v:%v", POST_INSTALL_SCRIPT_PATH_CM, ssh_user, ipv4s_str[0], POST_INSTALL_SCRIPT_PATH_VM)
	log.Printf("\t\t> %v\n", command)
	_, err = cm_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: CM SSH: Failed to copy post-install script to VM: %v\nOutput:\n%s", err, stdout)
	}

	//! Execute post-install script on VM
	log.Printf("\t[-] Executing post-install script on VM\n")

	log.Printf("\t\t[-] Running post-install script on VM\n")
	defer cm_sftp.Remove(POST_INSTALL_LOG_PATH_CM)
	command = fmt.Sprintf("ssh \"%v@%v\" \"chmod +x %v && sudo %v | sudo tee %v\"", ssh_user, ipv4s_str[0], POST_INSTALL_SCRIPT_PATH_VM, POST_INSTALL_SCRIPT_PATH_VM, POST_INSTALL_LOG_PATH_CM)
	log.Printf("\t\t> %v\n", command)
	stdout, err = cm_ssh.Run(command)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Failed to run post-install script on VM: %v\nStdout: %v", err, string(stdout))
	}

	_ = comp_node
	_ = example_fqdn
	_ = net
	_ = CEPH_POOL
	_ = VM_NETMASK_4
	_ = VM_GATEWAY_4
	_ = VM_NETMASK_6
	_ = VM_GATEWAY_6
	_ = TEMPLATE_STORAGE
	_ = TEMPLATE_STORAGE_ON_COMP
	_ = VM_SWAP_SIZE
	_ = VM_DEFAULT_ROOT_SIZE
	_ = VM_DEFAULT_RAM_SIZE
	_ = ssh_user
	_ = first_boot_line
	_ = IMAGE
	_ = IMAGE_REMOTE
	_ = SOURCES_LIST
	_ = VM_ID

	_ = VM_NETMODEL
	_ = IMAGE_PATH
	_ = FIRST_BOOT_LINE
	_ = VM_DESC
	_ = SWAP_SIZE
	_ = EFI_DISK_NAME
	_ = SWAP_DISK_NAME
	_ = MAIN_DISK_NAME

	//! Get VM data
	vm, err := GetNodeVM(comp_node_name, VM_ID)
	if err != nil {
		return nil, fmt.Errorf("Failed to create VM: Retrieving newly created VM failed: %v", err)
	}

	log.Println(`[+] Created VM ` + strconv.Itoa(vm.Vmid) + ` on node ` + comp_node_name + `
` + ssh_user + `@` + options.FQDN + `
IPv4[0]:` + ipv4s_str[0] + `
IPv6[0]:` + ipv6s_str[0] + `
Image: ` + options.Template + `
CPU: ` + strconv.FormatFloat(vm.Cpus, 'f', -1, 64) + `
RAM: ` + strconv.Itoa(vm.Maxmem) + `
Disk: ` + strconv.Itoa(vm.Maxdisk) + `
Fingerprints:
` + "\t" + strings.Join(vm_fingerprints, "\n\t"))

	// todo: add check if Org add to vsos-org instead
	err = AddVMToResourcePool(vm.Vmid, "vsos")
	return vm, err
}

func ExistsVMName(hostname string) (bool, error) {
	vms, err := GetAllClusterVMs()
	if err != nil {
		return false, fmt.Errorf("Failed to check if hostname '%v' is taken: %v", hostname, err.Error())
	}

	for _, m := range *vms {
		if m.Name == hostname {
			return true, nil
		}
	}

	return false, nil
}

func createSSHClient(pkey_path string, pkey_passphrase string, user string, hostname string) (*goph.Client, error) {
	auth, err := goph.Key(pkey_path, pkey_passphrase)
	if err != nil {
		return nil, fmt.Errorf("Failed to create SSH client: Failed to create pkey auth method: %v", err.Error())
	}

	client, err := goph.New(user, hostname, auth)
	if err != nil {
		return nil, fmt.Errorf("Failed to create SSH client: %v", err.Error())
	}

	return client, nil
}

func createCMSSHClient() (*goph.Client, error) {
	// Start new ssh connection with private key.
	client, err := createSSHClient("/root/.ssh/cm_pkey.key", config.AppConfig.SSH_CM_PKEY_PASSPHRASE, config.AppConfig.SSH_CM_USER, config.AppConfig.SSH_CM_HOST)
	if err != nil {
		return nil, fmt.Errorf("Failed to create CM node SSH client: %v", err.Error())
	}

	return client, nil
}

func createCompSSHClient() (*goph.Client, error) {
	// Start new ssh connection with private key.
	client, err := createSSHClient("/root/.ssh/comp_pkey.key", config.AppConfig.SSH_COMP_PKEY_PASSPHRASE, config.AppConfig.SSH_COMP_USER, config.AppConfig.SSH_COMP_HOST)
	if err != nil {
		return nil, fmt.Errorf("Failed to create CM node SSH client: %v", err.Error())
	}

	return client, nil
}

func createCMSFTPClient() (*sftp.Client, error) {
	sshclient, err := createCMSSHClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create CM node SFTP client: %v", err.Error())
	}

	sftpclient, err := sshclient.NewSftp()
	if err != nil {
		return nil, fmt.Errorf("Failed to create CM node SFTP client: %v", err.Error())
	}

	return sftpclient, nil
}

func createCompSFTPClient() (*sftp.Client, error) {
	sshclient, err := createCompSSHClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create Comp node SFTP client: %v", err.Error())
	}

	sftpclient, err := sshclient.NewSftp()
	if err != nil {
		return nil, fmt.Errorf("Failed to create Comp node SFTP client: %v", err.Error())
	}

	return sftpclient, nil
}

// TODO: Perform proper marshalling
func GetTokenPermissions() (string, error) {
	req, client, err := proxmoxMakeRequest("GET", "/api2/json/access/permissions", nil)
	if err != nil {
		return "", fmt.Errorf("Failed to get permissions: %v", err.Error())
	}

	body, err := proxmoxDoRequest(req, client)
	if err != nil {
		return "", fmt.Errorf("Failed to get permissions: %v", err.Error())
	}
	return string(body), nil
}

func TestCMConnection() error {
	client, err := createCMSSHClient()
	if err != nil {
		return fmt.Errorf("Testing CM connection: %v", err.Error())
	}
	_, err = client.Run("ls")
	if err != nil {
		return fmt.Errorf("Testing CM connection: %v", err.Error())
	}

	return nil
}

// PUT /api2/json/pools/{pool}
func AddVMToResourcePool(vm_id int, pool string) error {
	type bodyS struct {
		Vms int `json:"vms"`
	}
	body := bodyS{
		Vms: vm_id,
	}
	bodyB, err := json.Marshal(&body)
	if err != nil {
		return fmt.Errorf("Failed to add VM '%v' to resource pool '%v': %v", vm_id, pool, err.Error())
	}

	req, client, err := proxmoxMakeRequest(http.MethodPut, fmt.Sprintf("/api2/json/pools/%v", pool), bodyB)
	if err != nil {
		return fmt.Errorf("Failed to add VM '%v' to resource pool '%v': %v", vm_id, pool, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	q := req.URL.Query()

	req.URL.RawQuery = q.Encode()

	_, err = proxmoxDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Failed to add VM '%v' to resource pool '%v': %v", vm_id, pool, err.Error())
	}
	return nil
}
