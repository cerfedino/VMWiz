package main

import (
	"fmt"
	"regexp"

	"golang.org/x/exp/slices"
)

// The received form data
type Form struct {
	Email          string `json:"email"`
	Personal_email string `json:"personal_email"`
	IsOrganization bool   `json:"isOrganization"`
	OrgName        string `json:"orgName"`

	Hostname string `json:"hostname"`
	Image    string `json:"image"`
	Cores    int    `json:"cores"`
	Ram_gb   int    `json:"ram_gb"`
	Disk_gb  int    `json:"disk_gb"`

	Ssh_pubkey []string `json:"ssh_pubkey"`

	Comments     string `json:"comments"`
	Accept_terms bool   `json:"accept_terms"`
}

// The validation errors for the form
type Form_validation struct {
	Email_err          string `json:"email"`
	Personal_email_err string `json:"personal_email"`
	Hostname_err       string `json:"hostname"`
	Image_err          string `json:"image"`
	Cores_err          string `json:"cores"`
	Ram_gb_err         string `json:"ram_gb"`
	Disk_gb_err        string `json:"disk_gb"`

	Ssh_pubkey_err []string `json:"ssh_pubkey"`

	Accept_terms_err string `json:"accept_terms"`
}

// The allowed values for the form fields
type minmax struct {
	Min int `json:"min"`
	Max int `json:"max"`
}
type form_allowed_values struct {
	Images  []string `json:"image"`
	Cores   minmax   `json:"cores"`
	Ram_gb  minmax   `json:"ram_gb"`
	Disk_gb minmax   `json:"disk_gb"`
}

var ALLOWED_VALUES form_allowed_values = form_allowed_values{
	Images:  []string{"Ubuntu", "Debian"},
	Cores:   minmax{Min: 1, Max: 8},
	Ram_gb:  minmax{Min: 1, Max: 16},
	Disk_gb: minmax{Min: 1, Max: 100},
}

func (f *Form) toString() string {
	return fmt.Sprintf("\n  **Email**: %v\n **Personal Email**: %v\n **IsOrganization**: %v\n **OrgName**: %v\n **Hostname**: %v\n **Image**: %v\n **Cores**: %v\n **Ram**: %v\n **Disk**: %v\n **SSH Pubkey**: %v\n **Comments**: %v\n", f.Email, f.Personal_email, f.IsOrganization, f.OrgName, f.Hostname, f.Image, f.Cores, f.Ram_gb, f.Disk_gb, f.Ssh_pubkey, f.Comments)
}

func (f *Form) Validate() (Form_validation, bool) {
	var validation Form_validation
	var err bool = false

	university_email_regexp, _ := regexp.Compile("^[0-9A-Za-z+._~\\-!#$%&'.\\/=^{}|]+@(ethz|uzh)\\.ch$")
	if !university_email_regexp.Match([]byte(f.Email)) {
		validation.Email_err = "Must be a valid @ethz.ch or @uzh.ch email address"
		err = true
	}

	personal_email_regexp, _ := regexp.Compile("^[a-zA-Z0-9._%+-]+@([a-zA-Z0-9-]+\\.)+[a-zA-Z]{2,}$")
	if !personal_email_regexp.Match([]byte(f.Personal_email)) {
		validation.Personal_email_err = "Must be a valid email address"
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

	if f.Ram_gb < ALLOWED_VALUES.Ram_gb.Min {
		validation.Ram_gb_err = "Please select at least " + string(ALLOWED_VALUES.Ram_gb.Min) + " GB of RAM"
		err = true
	}

	if f.Disk_gb < ALLOWED_VALUES.Disk_gb.Min {
		validation.Disk_gb_err = "Please select at least " + string(ALLOWED_VALUES.Disk_gb.Min) + " GB of disk space"
		err = true
	}

	if len(f.Ssh_pubkey) == 0 {
		validation.Ssh_pubkey_err = []string{"Please provide at least one valid SSH public key"}
		err = true
	} else {
		ssh_pubkey_regexp, _ := regexp.Compile("^ssh-rsa [A-Za-z0-9+/=]+ [A-Za-z0-9+/.=]+")
		for _, key := range f.Ssh_pubkey {
			if !ssh_pubkey_regexp.Match([]byte(key)) {
				validation.Ssh_pubkey_err = append(validation.Ssh_pubkey_err, "Invalid SSH public key")
				err = true
			} else {
				validation.Ssh_pubkey_err = append(validation.Ssh_pubkey_err, "")
			}
		}
	}

	if !f.Accept_terms {
		validation.Accept_terms_err = "You must read and accept the terms"
		err = true
	}

	return validation, err
}
