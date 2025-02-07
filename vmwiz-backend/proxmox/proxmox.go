package proxmox

// https://pve.proxmox.com/wiki/Proxmox_VE_API#API_Tokens
// https://pve.proxmox.com/pve-docs/api-viewer/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"github.com/melbahja/goph"
	"github.com/pkg/sftp"
	"golang.org/x/exp/rand"
)

func proxmoxMakeRequest(method string, path string, body []byte) (*http.Request, *http.Client, error) {
	url, err := url.Parse(fmt.Sprintf("%v%v", os.Getenv("PVE_HOST"), path))
	if err != nil {
		return nil, nil, fmt.Errorf("Creating request: Parsing URL: %v", err.Error())
	}
	// log.Println("Requesting URL: '" + url.String() + "'")
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("Creating request %v %v: %v", method, url, err.Error())
	}

	req.Header.Add("Authorization", fmt.Sprintf("PVEAPIToken=%v!%v=%v", os.Getenv("PVE_USER"), os.Getenv("PVE_TOKENID"), os.Getenv("PVE_UUID")))

	return req, &http.Client{Timeout: time.Second * 10}, nil
}

func proxmoxDoRequest(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Making request %v %v: %v", req.Method, req.URL, err.Error())
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Making request: Cannot read body: %v", err.Error())
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Making request: Status %v\nBody: %v", res.Status, string(body))
	}

	return body, nil
}

// /api2/json/nodes
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

// /api2/json/cluster/resources?type=vm
type PVEVM struct {
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
}
type pveVMlist struct {
	Data []PVEVM `json:"data"`
}

func GetAllNodes() (*[]PVENode, error) {
	req, client, err := proxmoxMakeRequest(http.MethodGet, "/api2/json/nodes", []byte{})
	if err != nil {
		log.Println("Failed to retrieve all Proxmox nodes: %v", err.Error())
		return nil, err
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

func GetAllVMs() (*[]PVEVM, error) {
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

	var vms pveVMlist
	err = json.Unmarshal(body, &vms)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve all Proxmox VMs: Unmarshal error: %v", err.Error())
	}

	return &(vms.Data), nil
}

type PVEVMOptions struct {
	Template     string
	FQDN         string
	Reinstall    bool
	RAM_MB       int64
	Disk_GB      int64
	UseQemuAgent bool
	Description  string
	SSHPubkeys   []string
	nethz_user   string
	nethz_pass   string
}

func CreateVM(options PVEVMOptions) error {

	//! Verify that configured CH SSH host is actually a cluster management node
	// fmt.Println("[-] Checking if running on a cluster management node")
	client, err := createCMSSHClient()
	if err != nil {
		return fmt.Errorf("Failed to create VM: %v", err)
	}
	defer client.Close()

	log.Println("[-] Checking if CM SSH session is actually on a cluster management node")
	stdout, err := client.Run("hostname --fqdn")
	if err != nil {
		return fmt.Errorf("Failed to create VM: Cannot verify hostname of configured CM host: %v", err)
	}
	strstdout := strings.Trim(string(stdout), " \n")
	match, _ := regexp.MatchString("^cm-.+\\.sos\\.ethz\\.ch$", strstdout)
	if !match {
		return fmt.Errorf("Failed to create VM: Configured PVE SSH host is not a cluster management node")
	}

	//! Prepare default/hardcoded parameters

	comp_node := os.Getenv("SSH_COMP_HOST")
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

	template := options.Template
	ssh_user := "root"
	first_boot_line := "no"

	VMKEY_PATH := "/root/.ssh/vm_key.pub"

	//! Choosing appropriate user and first boot line
	log.Println("[-] Choosing appropriate user and first boot line based on template")
	switch template {
	case "bullseye":
		ssh_user = "debian"
		first_boot_line = "Cloud-init .* finished"
	case "bookworm":
		ssh_user = "debian"
		first_boot_line = "Cloud-init .* finished"
	case "jammy":
		ssh_user = "ubuntu"
		first_boot_line = "Cloud-init .* finished"
	case "noble":
		ssh_user = "ubuntu"
		first_boot_line = "Cloud-init .* finished"
	default:
		return fmt.Errorf("\t[X] Unknown template %v", template)
	}

	//! Checking existence of DNS entries for chosen FQDN
	log.Printf("[-] Checking existence of DNS entries for chosen FQDN %v", options.FQDN)
	ipv4s, ipv6s, err := netcenter.GetHostIPs(options.FQDN)
	if err != nil {
		return fmt.Errorf("Failed to create VM: PVE SSH host: Failed to check if there are existing ipv4 DNS entries for FQDN: %v", err.Error())
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

	if len(ipv4s_str) > 1 || len(ipv6s_str) > 1 {
		return fmt.Errorf("Failed to create VM: Each VM hostname should have AT MOST one IPv4 and one IPv6 address", options.FQDN)
	}

	if options.Reinstall && (len(ipv4s_str) == 0 || len(ipv6s_str) == 0) {
		return fmt.Errorf("Failed to create VM: Cannot reinstall VM with FQDN %v as it does not have both ipv4 and ipv6 DNS entries", options.FQDN)
	}

	if !options.Reinstall && (len(ipv4s_str) > 0 || len(ipv6s_str) > 0) {
		log.Println("\t[!] FQDN still has DNS entries with IP addresses:")
		// TODO: You sure you want to continue?
		return nil
	}

	//! Check if the image exists on the management node
	IMAGE := fmt.Sprintf("%v/cloudinit/current-%v-amd64.qcow2", TEMPLATE_STORAGE, options.Template)
	IMAGE_REMOTE := fmt.Sprintf("%v/cloudinit/current-%v-amd64.qcow2", TEMPLATE_STORAGE_ON_COMP, options.Template)

	log.Printf("[-] Checking if image '%v' exists on management node", IMAGE)

	cm_sftp, err := createCMSFTPClient()
	if err != nil {
		return fmt.Errorf("Failed to create VM: CM SFTP: %v", err.Error())
	}
	defer cm_sftp.Close()

	_, err = cm_sftp.Stat(IMAGE)
	if err != nil {
		return fmt.Errorf("Failed to create VM: Cannot ensure existence of '%v' on CM node: %v", IMAGE, err)
	}

	//! Preparing sources.list for VM
	log.Println("[-] Preparing apt sources for VM")
	var SOURCES_LIST string
	if options.Template == "bullseye" || options.Template == "bookworm" {
		SOURCES_LIST = `
		deb http://ftp.ch.debian.org/debian $template main
		#deb-src http://ftp.ch.debian.org/debian $template main

		deb http://ftp.ch.debian.org/debian $template-updates main
		#deb-src http://ftp.ch.debian.org/debian $template-updates main

		deb http://security.debian.org/ $template-security main
		#deb-src http://security.debian.org/ $template-security main`
	} else if options.Template == "jammy" || options.Template == "noble" {
		SOURCES_LIST = `
		deb http://ch.archive.ubuntu.com/ubuntu $template main universe multiverse
		#deb-src http://ch.archive.ubuntu.com/ubuntu $template main universe multiverse

		deb http://ch.archive.ubuntu.com/ubuntu $template-updates main universe multiverse
		#deb-src http://ch.archive.ubuntu.com/ubuntu $template-updates main universe multiverse

		deb http://security.ubuntu.com/ubuntu $template-security main universe multiverse
		#deb-src http://security.ubuntu.com/ubuntu $template-security main universe multiverse`
	} else {
		return fmt.Errorf("Failed to create VM: Unknown template %v", options.Template)
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
RAM: %v
Disk size: %v
Swap size: %v
Ceph pool: %v

QEMU agent: %v

Reinstall: %v
-------
`, VM_ID, options.FQDN, options.Description, options.Template, options.RAM_MB, options.Disk_GB, VM_SWAP_SIZE, CEPH_POOL, options.UseQemuAgent, options.Reinstall)

	//! Register DNS entries for FQDN and an available IPv4 and IPv6 address.
	if !options.Reinstall {
		log.Printf("[-] Registering \"FQDN\" %v in net \"%v\"\n", options.FQDN, net)
		ipv4, ipv6, err := netcenter.Registerhost(net, options.FQDN)
		if err != nil {
			return fmt.Errorf("Failed to create VM: %v", err)
		}
		ipv4s_str = append(ipv4s_str, (*ipv4).String())
		ipv6s_str = append(ipv6s_str, (*ipv6).String())
	}

	//! Read universal VM public key from CM LEE
	// TODO: Startup check
	// TODO: Have the default universal public key of the VMs stored in the configs.
	log.Println("[-] Reading universal VM public key from CM LEE")

	vmkey_file, err := cm_sftp.Open(VMKEY_PATH)
	if err != nil {
		return fmt.Errorf("Failed to create VM: PVE SFTP: Failed to open the VM key '%v': %v", VMKEY_PATH, err)
	}
	defer vmkey_file.Close()

	vmkey_content, err := io.ReadAll(vmkey_file)
	if err != nil {
		return fmt.Errorf("Failed to create VM: PVE SFTP: Failed to read the VM key '%v': %v", VMKEY_PATH, err)
	}

	//! Prepare authorized_keys file
	log.Println("[-] Preparing authorized_keys file\n\tConcatenating VM universal public key with provided pubkeys")
	authorized_keys_content := strings.Join(slices.Concat(options.SSHPubkeys, strings.Split(string(vmkey_content), "\n")), "\n\n")

	//! Upload authorized_keys file to comp node
	VM_AUTHORIZED_KEYS_PATH := fmt.Sprintf("/tmp/%v.ssh.pub", VM_ID)
	log.Printf("[-] Uploading authorized_keys file to %v:%v\n", comp_node, VM_AUTHORIZED_KEYS_PATH)
	comp_sftp, err := createCompSFTPClient()
	if err != nil {
		return fmt.Errorf("Failed to create VM: Comp node SFTP: %v", err.Error())
	}

	comp_sftp_authorized_keys, err := comp_sftp.Create(VM_AUTHORIZED_KEYS_PATH)
	if err != nil {
		return fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to create file '%v': %v", VM_AUTHORIZED_KEYS_PATH, err)
	}
	defer comp_sftp_authorized_keys.Close()
	defer comp_sftp.Remove(VM_AUTHORIZED_KEYS_PATH)

	_, err = comp_sftp_authorized_keys.Write([]byte(authorized_keys_content))
	if err != nil {
		return fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to write to file '%v': %v", VM_AUTHORIZED_KEYS_PATH, err)
	}

	//! Prepare Cloudinit configuration
	log.Println("[-] Preparing Cloudinit configuration")
	cloudinit_fragments := fmt.Sprintf("ipconfig0: gw=%s,ip=%s/%d,ip6=%s/%d", VM_GATEWAY_4, ipv4s_str[0], VM_NETMASK_4, ipv6s_str[0], VM_NETMASK_6)
	// fmt.Println(cloudinit_fragments)

	//! Upload Cloudinit configuration to comp node
	VM_CLOUDINIT_PATH := fmt.Sprintf("/tmp/%v.pve.tail", VM_ID)
	log.Printf("[-] Uploading Cloudinit fragments to %v:%v\n", comp_node, VM_CLOUDINIT_PATH)
	comp_sftp_cloudinitfrags, err := comp_sftp.Create(VM_CLOUDINIT_PATH)
	if err != nil {
		return fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to create file '%v': %v", VM_CLOUDINIT_PATH, err)
	}
	defer comp_sftp_cloudinitfrags.Close()
	defer comp_sftp.Remove(VM_CLOUDINIT_PATH)

	_, err = comp_sftp_cloudinitfrags.Write([]byte(cloudinit_fragments))
	if err != nil {
		return fmt.Errorf("Failed to create VM: Comp node SFTP: Failed to write to file '%v': %v", VM_CLOUDINIT_PATH, err)
	}

	//!

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
	return nil
}

func IsHostnameTaken(hostname string) (bool, error) {
	// TODO: We check only if there is a VM with the same name. Should we also check DNS ?
	vms, err := GetAllVMs()
	if err != nil {
		return false, fmt.Errorf("Failed to check if hostname '%v' is taken: %v", err.Error())
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
	client, err := createSSHClient("/root/.ssh/cm_pkey.key", os.Getenv("SSH_CM_PKEY_PASSPHRASE"), os.Getenv("SSH_CM_USER"), os.Getenv("SSH_CM_HOST"))
	if err != nil {
		return nil, fmt.Errorf("Failed to create CM node SSH client: %v", err.Error())
	}

	return client, nil
}

func createCompSSHClient() (*goph.Client, error) {
	// Start new ssh connection with private key.
	client, err := createSSHClient("/root/.ssh/comp_pkey.key", os.Getenv("SSH_COMP_PKEY_PASSPHRASE"), os.Getenv("SSH_COMP_USER"), os.Getenv("SSH_COMP_HOST"))
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
