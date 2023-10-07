package mailer

import (
    "bytes"
    "embed"
    "html/template"
    "time"

    "github.com/go-mail/mail/v2"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
    dialer *mail.Dialer
    sender string
}

func New(host string, port int, username, password, sender string) Mailer {
    dialer := mail.NewDialer(host, port, username, password)
    dialer.Timeout = 5 * time.Second

    return Mailer{
        dialer: dialer,
        sender: sender,
    }
}

func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
    tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
    if err != nil {
        return err
    }

    subject := new(bytes.Buffer)
    err = tmpl.ExecuteTemplate(subject, "subject", data)
    if err != nil {
        return err
    }

    plainBody := new(bytes.Buffer)
    err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
    if err != nil {
        return err
    }

    htmlBody := new(bytes.Buffer)
    err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
    if err != nil {
        return err
    }

    message := mail.NewMessage()
    message.SetHeader("To", recipient)
    message.SetHeader("From", m.sender)
    message.SetHeader("Subject", subject.String())
    message.SetBody("text/plain", plainBody.String())
    message.AddAlternative("text/html", htmlBody.String())

    for i := 1; i <= 3; i++ {
        err = m.dialer.DialAndSend(message)
        if nil == err {
            return nil
        }
        time.Sleep(time.Millisecond*500)
    }

    return nil
}
