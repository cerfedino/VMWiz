package survey

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"strings"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/google/uuid"
)

type SurveyVM struct {
	Nethz            string
	University_email string
	ExternalMail     string
	Hostname         string
	VMID             int
}

func CreateVMUsageSurvey(restrict_pool []string) error {
	vms, err := generateSurveys(restrict_pool)
	if err != nil {
		return err
	}

	surveyID, err := storage.DB.SurveyStore()
	if err != nil {
		return err
	}

	err = notifier.NotifyVMUsageSurvey(surveyID, fmt.Sprintf("Created new VM usage survey with ID %d", surveyID))
	if err != nil {
		return fmt.Errorf("Failed create VM usage survey: %v", err)
	}

	emails_sent := 0
	for _, vm := range vms {
		if emails_sent%10 == 0 {
			log.Printf("Sending emails ... (%v / %v)", emails_sent, len(vms))
		}

		uuidString := uuid.New().String()

		_, err := storage.DB.SurveyQuestionStore(surveyID, vm.VMID, vm.Hostname, uuidString)
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: %v", err)
		}

		// Receiver email address.
		receivers := []string{}
		if config.AppConfig.SMTP_RECEIVER_OVERRIDE != "" {
			// Override the receiver email address with the one from the config if present
			receivers = []string{config.AppConfig.SMTP_RECEIVER_OVERRIDE}
		} else {
			receivers = []string{vm.University_email}
		}

		VMUSAGE_SURVEY_TEMPLATE_PATH := "survey/vmusage_survey.tmpl"
		mail_content := new(bytes.Buffer)
		vmusage_survey_template, err := template.ParseFiles(VMUSAGE_SURVEY_TEMPLATE_PATH)
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: Failed to parse email template: %v", err)
		}
		err = vmusage_survey_template.Execute(mail_content, struct {
			HOSTNAME string
			URL      string
		}{
			HOSTNAME: vm.Hostname,
			URL:      config.AppConfig.VMWIZ_SCHEME + "://" + config.AppConfig.VMWIZ_HOSTNAME + ":" + strconv.Itoa(config.AppConfig.VMWIZ_PORT) + "/survey?id=" + uuidString,
		})
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: Failed to execute email template: %v", err)
		}

		if !config.AppConfig.SMTP_ENABLE {
			emails_sent++
			continue
		}

		// Authentication.
		// fmt.Println(config.AppConfig.SMTP_USER, config.AppConfig.SMTP_PASSWORD, config.AppConfig.SMTP_HOST, config.AppConfig.SMTP_PORT)
		// TODO: Add startup check
		err = notifier.SendEmail("VSOS VM Usage Survey: Response needed", mail_content.Bytes(), receivers)
		if err != nil {
			return fmt.Errorf("Failed create VM usage survey: Failed to send email: %v", err)
		}
		emails_sent++
	}

	var msg string
	if config.AppConfig.SMTP_ENABLE {
		msg = fmt.Sprintf("Sent %d emails for VM usage survey %v", emails_sent, surveyID)
	} else {
		msg = fmt.Sprintf("Dry-run Sent %d emails for VM usage survey %v(SMTP disabled)", emails_sent, surveyID)
	}
	err = notifier.NotifyVMUsageSurvey(surveyID, msg)
	if err != nil {
		return fmt.Errorf("Failed to send VM usage survey notification: %v", err)
	}
	log.Printf("[+] " + msg)
	return nil
}

func generateSurveys(restrict_pool []string) ([]SurveyVM, error) {
	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		return nil, fmt.Errorf("Failed to get VM list: %v", err.Error())
	}

	surveyList := make([]SurveyVM, 0)

	for _, m := range *vms {
		// If restrict_pool is not empty, check if the VM is is one of the allowed pools
		if len(restrict_pool) > 0 {
			found := false
			for _, pool := range restrict_pool {
				if m.Pool == pool {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		vmConfig, err := proxmox.GetNodeVMConfig(m.Node, m.Vmid)
		if err != nil {
			continue
		}

		nethz, err := getDescriptionField(vmConfig.Description, "nethz=")
		if err != nil {
			continue
		}
		mail, err := getDescriptionField(vmConfig.Description, "uni_contact=")
		if err != nil {
			continue
		}
		externalMail, err := getDescriptionField(vmConfig.Description, "contact=")
		if err != nil {
			continue
		}

		vm := SurveyVM{
			Hostname:         m.Name,
			VMID:             m.Vmid,
			Nethz:            nethz,
			University_email: mail,
			ExternalMail:     externalMail,
		}
		surveyList = append(surveyList, vm)
	}

	return surveyList, nil
}

func getDescriptionField(description string, field string) (string, error) {
	lines := strings.Split(description, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, field) {
			return strings.TrimSpace(strings.TrimPrefix(line, field)), nil
		}
	}
	return "", fmt.Errorf("Field '%s' not found in description", field)
}
