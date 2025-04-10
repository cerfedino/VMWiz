package survey

import (
	"fmt"
	"net/smtp"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/google/uuid"
)

// this needs to get all VMs from VSOS pool
// create a unique id for each VM
// save that id with VM id in DB

// send mail to user with url+id

// .get("/cluster/resources")
// .get(format!("/nodes/{}/qemu/{}/config", self.node, self.vmid))

func SendSurvey() error {
	vms, err := proxmox.GetMailAndId()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		// send mail to user with url+uuid
		// create uniqe uuid
		id := uuid.New()
		uuidString := id.String()
		// save id with vm id in DB
		surveyId, err := storage.DB.AddSurvey()
		if err != nil {
			return err
		}
		err = storage.DB.StoreSurveyId(vm.VMID, vm.Hostname, surveyId, uuidString)
		if err != nil {
			return err
		}

		url := "https://vmwiz.vsos.ethz.ch/survey?id=" + uuidString

		smtpHost := "mail.sos.ethz.ch"
		smtpPort := "587"
		//todo: actual account / password
		sender := "vm-wizard@sos.ethz.ch"
		password := "your-password"

		// Receiver email address.
		// to := []string{vm.Mail}
		to := []string{""} //todo: change to actual email addresses and not a test one

		// Message.
		subject := "Subject: VSOS VM Usage Survey: Response needed\n"
		body := "You have a VM with us: " + vm.Hostname + "\n" +
			"Please state if you still use your VM." + url + "\n" +
			"If you do not fill out the link we will send follow up mails and shutdown your VM.\n"
		message := []byte(subject + "\n" + body)

		// Authentication.
		auth := smtp.PlainAuth("", sender, password, smtpHost)

		// Sending email.
		err = smtp.SendMail(smtpHost+":"+smtpPort, auth, sender, to, message)
		if err != nil {
			fmt.Println("Error sending email:", err)
			return err
		}
		fmt.Println("Email sent successfully")
	}
	return nil
}
