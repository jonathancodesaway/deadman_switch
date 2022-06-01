package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"sync"
	"time"
)

const (
	hoursOneWeek  = 168
	hoursOneMonth = hoursOneWeek * 7
	sender        = "sender@gmail.com"
	// using app password feature for gmail
	appPassword = "appPassword"

	// smtp server configuration.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)

var (
	msg = []byte("To: receiver@gmail.com\r\n" +
		"Subject: are you dead yet?\r\n" +
		"\r\n" +
		"Please visit this link to reset deadman switch: http://www.localhost:8080/reset_deadman\r\n")

	alertRecipents = []string{
		"receiver@gmail.com",
	}

	// for resiliency, instead of storing in RAM, store in a distributed data store.
	lastDeadmanVisit = time.Now()
)

func deadmanAlert() error {

	// Authentication.
	auth := smtp.PlainAuth("", sender, appPassword, smtpHost)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, sender, alertRecipents, msg)
	if err != nil {
		return err
	}
	log.Print("email sent")
	return nil
}

func deliverDeadmanPayload() {
	// send an email similar to alert. perhaps a text file?
	fmt.Println("i'm dead")
}

func resetDeadman(w http.ResponseWriter, req *http.Request) {
	lastDeadmanVisit = time.Now()
	log.Printf("setting last deadman time to %q\n", lastDeadmanVisit)
	io.WriteString(w, fmt.Sprintf("Deadman is reset to %q\n", lastDeadmanVisit))
}

func startHttpServer(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/reset_deadman", resetDeadman)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return srv
}

func main() {
	log.Printf("main: starting HTTP server")
	ctx := context.Background()

	var httpServerExitDone sync.WaitGroup
	httpServerExitDone.Add(1)
	srv := startHttpServer(&httpServerExitDone)

	weeklyTicker := time.NewTicker(hoursOneWeek * time.Hour)
	for {
		t := <-weeklyTicker.C

		if diff := t.Sub(lastDeadmanVisit); diff > (hoursOneMonth * time.Hour) {
			deliverDeadmanPayload()
			break
		}
		if err := deadmanAlert(); err != nil {
			// quiet shut down. don't want to alert anything.
			// TODO: add maybe texting as a backup?
			break
		}
	}

	log.Printf("main: stopping HTTP server")
	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
	httpServerExitDone.Wait()
	log.Printf("main: stopped HTTP server")
}
