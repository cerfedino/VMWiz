package form

import (
	"fmt"
	"regexp"

	"golang.org/x/exp/slices"
)

// The received form data
type Form struct {
	Email          string `json:"email"`
	PersonalEmail  string `json:"personalEmail"`
	IsOrganization bool   `json:"isOrganization"`
	OrgName        string `json:"orgName"`

	Hostname string `json:"hostname"`
	Image    string `json:"image"`
	Cores    int    `json:"cores"`
	RamGB    int    `json:"ramGB"`
	DiskGB   int    `json:"diskGB"`

	SshPubkeys []string `json:"sshPubkey"`

	Comments     string `json:"comments"`
	Accept_terms bool   `json:"accept_terms"`
}

// The validation errors for the form
type Form_validation struct {
	Email_err         string `json:"email"`
	PersonalEmail_err string `json:"personalEmail"`
	OrgName_err       string `json:"orgName"`
	Hostname_err      string `json:"hostname"`
	Image_err         string `json:"image"`
	Cores_err         string `json:"cores"`
	RamGB_err         string `json:"ramGB"`
	DiskGB_err        string `json:"diskGB"`

	SshPubkeys_err []string `json:"sshPubkey"`

	Accept_terms_err string `json:"accept_terms"`
}

// The allowed values for the form fields
type minmax struct {
	Min int `json:"min"`
	Max int `json:"max"`
}
type form_allowed_values struct {
	Images []string `json:"image"`
	Cores  minmax   `json:"cores"`
	RamGB  minmax   `json:"ramGB"`
	DiskGB minmax   `json:"diskGB"`
}

var ALLOWED_VALUES form_allowed_values = form_allowed_values{
	Images: []string{"Ubuntu", "Debian"},
	Cores:  minmax{Min: 1, Max: 8},
	RamGB:  minmax{Min: 1, Max: 16},
	DiskGB: minmax{Min: 1, Max: 100},
}

func (f *Form) ToString() string {
	return fmt.Sprintf("\n  **Email**: %v\n **Personal Email**: %v\n **IsOrganization**: %v\n **OrgName**: %v\n **Hostname**: %v\n **Image**: %v\n **Cores**: %v\n **Ram**: %v\n **Disk**: %v\n **SSH Pubkey**: %v\n **Comments**: %v\n", f.Email, f.PersonalEmail, f.IsOrganization, f.OrgName, f.Hostname, f.Image, f.Cores, f.RamGB, f.DiskGB, f.SshPubkeys, f.Comments)
}

func (f *Form) Validate() (Form_validation, bool) {
	var validation Form_validation
	var err bool = false

	university_email_regexp, _ := regexp.Compile("^[0-9A-Za-z+._~\\-!#$%&'.\\/=^{}|]+@(ethz|uzh)\\.ch$")
	if !university_email_regexp.Match([]byte(f.Email)) {
		validation.Email_err = "Must be a valid @ethz.ch or @uzh.ch email address"
		err = true
	}

	personalEmail_regexp, _ := regexp.Compile("^[a-zA-Z0-9._%+-]+@([a-zA-Z0-9-]+\\.)+[a-zA-Z]{2,}$")
	if !personalEmail_regexp.Match([]byte(f.PersonalEmail)) {
		validation.PersonalEmail_err = "Must be a valid email address"
		err = true
	}

	if f.IsOrganization && f.OrgName == "" {
		validation.OrgName_err = "Must be a valid organization name"
		err = true
	}

	if len(f.Hostname) == 0 {
		validation.Hostname_err = "Hostname cannot be empty"
		err = true
	}

	hostname_regexp, _ := regexp.Compile("^[a-zA-Z0-9-]+$")
	if !hostname_regexp.Match([]byte(f.Hostname)) {
		validation.Hostname_err = "Must be a valid hostname"
		// TODO: Check if hostname is already taken
		err = true
	}

	if f.Image == "" {
		validation.Image_err = "Please specify an image"
	}

	// If f image no in list of images
	if !slices.Contains(ALLOWED_VALUES.Images, f.Image) {
		validation.Image_err = "Please select a valid image"
		err = true
	}

	if f.Cores < ALLOWED_VALUES.Cores.Min {
		validation.Cores_err = "Please select at least " + string(ALLOWED_VALUES.Cores.Min) + " cores"
		err = true
	}

	if f.RamGB < ALLOWED_VALUES.RamGB.Min {
		validation.RamGB_err = "Please select at least " + string(ALLOWED_VALUES.RamGB.Min) + " GB of RAM"
		err = true
	}

	if f.DiskGB < ALLOWED_VALUES.DiskGB.Min {
		validation.DiskGB_err = "Please select at least " + string(ALLOWED_VALUES.DiskGB.Min) + " GB of disk space"
		err = true
	}

	if len(f.SshPubkeys) == 0 {
		validation.SshPubkeys_err = []string{"Please provide at least one valid SSH public key"}
		err = true
	} else {
		sshPubkey_regexp, _ := regexp.Compile("^ssh-rsa [A-Za-z0-9+/=]+ [A-Za-z0-9+/.=]+")
		for _, key := range f.SshPubkeys {
			if !sshPubkey_regexp.Match([]byte(key)) {
				validation.SshPubkeys_err = append(validation.SshPubkeys_err, "Invalid SSH public key")
				err = true
			} else {
				validation.SshPubkeys_err = append(validation.SshPubkeys_err, "")
			}
		}
	}

	if !f.Accept_terms {
		validation.Accept_terms_err = "You must read and accept the terms"
		err = true
	}

	return validation, err
}
