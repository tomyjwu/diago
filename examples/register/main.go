// SPDX-License-Identifier: MPL-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024, Emir Aganovic

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/emiago/diago"
	"github.com/emiago/diago/examples"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// Run app:
// go run . sip:user@myregistrar.com
// go run . -username alice -password secret -proxy 192.168.1.100:5060 sip:alice@example.com

func main() {
	fUsername := flag.String("username", "", "Digest username")
	fPassword := flag.String("password", "", "Digest password")
	fProxy := flag.String("proxy", "", "Outbound proxy host:port (e.g., 192.168.1.100:5060)")
	fDebug := flag.Bool("debug", false, "Enable debug logging")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -username <username> -password <pass> [-proxy <proxy>] [-debug] sip:123@example.com\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -username alice -password secret sip:alice@example.com\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -username alice -password secret -proxy 192.168.1.100:5060 -debug sip:alice@example.com\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Setup signaling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Setup logger with debug level if requested
	if *fDebug {
		os.Setenv("LOG_LEVEL", "DEBUG")
		os.Setenv("SIP_DEBUG", "true")
	}
	examples.SetupLogger()

	recipientUri := flag.Arg(0)
	if recipientUri == "" {
		flag.Usage()
		return
	}

	regOpts := diago.RegisterOptions{
		Username:  *fUsername,
		Password:  *fPassword,
		ProxyHost: *fProxy,
		Expiry:    3600, // 1 hour
	}

	// Log registration configuration
	slog.Info("Starting registration",
		"recipient_uri", recipientUri,
		"username", regOpts.Username,
		"proxy_host", regOpts.ProxyHost,
		"expiry_seconds", int(regOpts.Expiry.Seconds()),
		"debug_enabled", *fDebug,
	)

	err := start(ctx, recipientUri, regOpts)
	if err != nil {
		slog.Error("Registration finished with error", "error", err)
		os.Exit(1)
	}
}

func start(ctx context.Context, recipientURI string, regOpts diago.RegisterOptions) error {
	recipient := sip.Uri{}
	if err := sip.ParseUri(recipientURI, &recipient); err != nil {
		return fmt.Errorf("failed to parse register uri: %w", err)
	}

	// Setup our main transaction user
	useragent := regOpts.Username
	if useragent == "" {
		useragent = "diago-register"
	}

	ua, _ := sipgo.NewUA(
		sipgo.WithUserAgent(useragent),
		sipgo.WithUserAgentHostname("localhost"),
	)
	defer ua.Close()

	// Configure transport with external host for NAT scenarios
	tu := diago.NewDiago(ua, diago.WithTransport(
		diago.Transport{
			Transport:    "udp",
			BindHost:     "127.0.0.1",
			BindPort:     15060,
			ExternalHost: "192.168.1.50", // Example external IP for NAT
			ExternalPort: 15060,
		},
	))

	// Start listening incoming calls
	go func() {
		tu.Serve(ctx, func(inDialog *diago.DialogServerSession) {
			slog.Info("New dialog request", "id", inDialog.ID)
			defer slog.Info("Dialog finished", "id", inDialog.ID)
		})
	}()

	// Do register or fail on error
	slog.Info("Starting registration process",
		"recipient", recipient.String(),
		"proxy", regOpts.ProxyHost,
	)

	return tu.Register(ctx, recipient, regOpts)
}
