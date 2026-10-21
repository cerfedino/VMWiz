package proxmox

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// HDDPool is the Ceph/Proxmox storage pool used for secondary (HDD) disks.
const HDDPool = "vm-norm"

// parseDiskSizeGB extracts the size of a Proxmox disk config string in GB.
// The string looks like "vm-norm:vm-100-disk-1,size=50G,..."; the size unit
// can be M, G, or T. Returns 0 if no size can be parsed.
func parseDiskSizeGB(disk string) int {
	for _, part := range strings.Split(disk, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || kv[0] != "size" {
			continue
		}
		raw := strings.TrimSpace(kv[1])
		if raw == "" {
			return 0
		}
		unit := raw[len(raw)-1]
		num, err := strconv.Atoi(raw[:len(raw)-1])
		if err != nil {
			return 0
		}
		switch unit {
		case 'T':
			return num * 1024
		case 'G':
			return num
		case 'M':
			return num / 1024
		}
		return 0
	}
	return 0
}

// SecondaryDiskGB returns the size in GB of scsi1, or 0 if the VM has none.
func (cfg *PVENodeVMConfig) SecondaryDiskGB() int {
	if cfg.Scsi1 == "" {
		return 0
	}
	return parseDiskSizeGB(cfg.Scsi1)
}

// SetVMCoresAndMemory updates the cores and memory of a VM via the Proxmox
// HTTP API. Memory is in MB. Either parameter is skipped if zero.
func SetVMCoresAndMemory(node string, vmid int, cores int, memoryMB int) error {
	form := url.Values{}
	if cores > 0 {
		form.Set("cores", strconv.Itoa(cores))
	}
	if memoryMB > 0 {
		form.Set("memory", strconv.Itoa(memoryMB))
	}
	if len(form) == 0 {
		return nil
	}

	body := []byte(form.Encode())
	req, client, err := proxmoxMakeRequest(http.MethodPut, fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/config", node, vmid), body)
	if err != nil {
		return fmt.Errorf("Failed to set VM cores/memory: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err = proxmoxDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Failed to set VM cores/memory: %v", err)
	}
	return nil
}

// ResizeVMDisk grows the main disk (scsi0) to the given target size in GB.
// Proxmox does not support shrinking, so this returns an error if the target
// is not strictly larger than the current size.
func ResizeVMDisk(node string, vmid int, currentGB int, targetGB int) error {
	return resizeVMDiskSlot(node, vmid, "scsi0", currentGB, targetGB)
}

// ResizeVMSecondaryDisk grows the secondary HDD disk (scsi1) to the given
// target size in GB. Same shrink restriction as the primary disk.
func ResizeVMSecondaryDisk(node string, vmid int, currentGB int, targetGB int) error {
	return resizeVMDiskSlot(node, vmid, "scsi1", currentGB, targetGB)
}

func resizeVMDiskSlot(node string, vmid int, slot string, currentGB, targetGB int) error {
	if targetGB == currentGB {
		return nil
	}
	if targetGB < currentGB {
		return fmt.Errorf("Cannot shrink VM disk %s (current %dG, target %dG)", slot, currentGB, targetGB)
	}

	form := url.Values{}
	form.Set("disk", slot)
	form.Set("size", fmt.Sprintf("%dG", targetGB))

	body := []byte(form.Encode())
	req, client, err := proxmoxMakeRequest(http.MethodPut, fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/resize", node, vmid), body)
	if err != nil {
		return fmt.Errorf("Failed to resize VM disk %s: %v", slot, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err = proxmoxDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Failed to resize VM disk %s: %v", slot, err)
	}
	return nil
}

// CreateVMSecondaryDisk attaches a new HDD (scsi1) to the VM with the given
// size. Errors if scsi1 is already populated.
func CreateVMSecondaryDisk(node string, vmid int, sizeGB int) error {
	if sizeGB <= 0 {
		return fmt.Errorf("Cannot create secondary disk with size %d", sizeGB)
	}

	cfg, err := GetNodeVMConfig(node, vmid)
	if err != nil {
		return fmt.Errorf("read VM config before HDD creation: %w", err)
	}
	if cfg.Scsi1 != "" {
		return fmt.Errorf("Cannot create secondary disk: scsi1 is already occupied")
	}
	if cfg.Digest == "" {
		return fmt.Errorf("Cannot create secondary disk without a VM config digest")
	}

	form := url.Values{}
	form.Set("digest", cfg.Digest)
	form.Set("scsi1", fmt.Sprintf("%s:%d", HDDPool, sizeGB))

	body := []byte(form.Encode())
	req, client, err := proxmoxMakeRequest(http.MethodPut, fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/config", node, vmid), body)
	if err != nil {
		return fmt.Errorf("Failed to attach secondary disk: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err = proxmoxDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Failed to attach secondary disk: %v", err)
	}
	return nil
}
