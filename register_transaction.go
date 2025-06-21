// SPDX-License-Identifier: MPL-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024, Emir Aganovic

package diago

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

type RegisterResponseError struct {
	RegisterReq *sip.Request
	RegisterRes *sip.Response

	Msg string
}

func (e *RegisterResponseError) StatusCode() int {
	return e.RegisterRes.StatusCode
}

func (e RegisterResponseError) Error() string {
	return e.Msg
}

type RegisterTransaction struct {
	opts   RegisterOptions
	Origin *sip.Request

	client *sipgo.Client
	log    *slog.Logger

	expiry time.Duration
}

func newRegisterTransaction(client *sipgo.Client, recipient sip.Uri, contact sip.ContactHeader, opts RegisterOptions) *RegisterTransaction {
	expiry, allowHDRS := opts.Expiry, opts.AllowHeaders
	// log := p.getLoggerCtx(ctx, "Register")
	req := sip.NewRequest(sip.REGISTER, recipient)
	req.AppendHeader(&contact)

	if opts.ProxyHost != "" {
		req.SetDestination(opts.ProxyHost)
	}
	if expiry > 0 {
		expires := sip.ExpiresHeader(expiry.Seconds())
		req.AppendHeader(&expires)
	}
	if allowHDRS != nil {
		req.AppendHeader(sip.NewHeader("Allow", strings.Join(allowHDRS, ", ")))
	}

	// Add custom headers if provided
	if opts.Headers != nil {
		for _, header := range opts.Headers {
			req.AppendHeader(header)
		}
	}

	// if opts.Username == "" {
	// 	opts.Username = opts.UserAgent
	// }

	if opts.Username == "" {
		opts.Username = client.Name()
	}

	t := &RegisterTransaction{
		Origin: req, // origin maybe updated after first register
		opts:   opts,
		client: client,
		log:    slog.Default().With("caller", "Register"),
	}

	// Debug: Log packet preparation details
	t.log.Debug("Register packet prepared",
		"recipient_uri", recipient.String(),
		"contact_uri", contact.Address.String(),
		"proxy_host", opts.ProxyHost,
		"expiry_seconds", int(expiry.Seconds()),
		"allow_headers", allowHDRS,
		"username", opts.Username,
		"request_method", req.Method.String(),
		"request_uri", req.Recipient.String(),
		"destination", req.Destination(),
	)

	return t
}

func (t *RegisterTransaction) Register(ctx context.Context) error {
	username, password, expiry := t.opts.Username, t.opts.Password, t.opts.Expiry
	client := t.client
	log := t.log
	req := t.Origin
	contact := *req.Contact().Clone()

	// Debug: Log connection preparation details
	log.Debug("Preparing to send REGISTER request",
		"client_name", client.Name(),
		"request_uri", req.Recipient.String(),
		"destination", req.Destination(),
		"contact_uri", contact.Address.String(),
		"proxy_host", t.opts.ProxyHost,
		"domain_vs_proxy", fmt.Sprintf("domain=%s, proxy=%s", req.Recipient.Host, t.opts.ProxyHost),
	)

	// Send request and parse response
	// req.SetDestination(*dst)
	log.Info("REGISTER", "uri", req.Recipient.String(), "expiry", int(expiry))

	// Debug: Log before sending packet
	log.Debug("Sending REGISTER packet via client.Do",
		"request_start_line", req.StartLine(),
		"via_header", req.Via(),
		"from_header", req.From(),
		"to_header", req.To(),
		"contact_header", req.Contact(),
		"expires_header", req.GetHeader("Expires"),
		"allow_header", req.GetHeader("Allow"),
	)

	res, err := client.Do(ctx, req, sipgo.ClientRequestRegisterBuild)
	if err != nil {
		log.Error("Failed to send REGISTER packet",
			"error", err,
			"request_start_line", req.StartLine(),
		)
		return fmt.Errorf("fail to create transaction req=%q: %w", req.StartLine(), err)
	}

	// Debug: Log successful packet sending
	log.Debug("REGISTER packet sent successfully",
		"response_status", res.StatusCode,
		"response_reason", res.Reason,
		"response_via", res.Via(),
	)

	via := res.Via()
	if via == nil {
		return fmt.Errorf("no Via header in response")
	}

	// https://datatracker.ietf.org/doc/html/rfc3581#section-9
	if rport, _ := via.Params.Get("rport"); rport != "" {
		if p, err := strconv.Atoi(rport); err == nil {
			contact.Address.Port = p
		}

		if received, _ := via.Params.Get("received"); received != "" {
			// TODO: consider parsing IP
			contact.Address.Host = received
		}

		// Update contact address of NAT
		req.ReplaceHeader(&contact)

		// Debug: Log NAT traversal updates
		log.Debug("NAT traversal updates applied",
			"rport", rport,
			"received", func() string {
				if received, _ := via.Params.Get("received"); received != "" {
					return received
				}
				return ""
			}(),
			"updated_contact", contact.Address.String(),
		)
	}

	log.Info("Received status", "status", int(res.StatusCode))
	if res.StatusCode == sip.StatusUnauthorized || res.StatusCode == sip.StatusProxyAuthRequired {
		log.Info("Unathorized. Doing digest auth")

		// Debug: Log digest auth attempt
		log.Debug("Starting digest authentication",
			"username", username,
			"auth_header", res.GetHeader("WWW-Authenticate"),
		)

		res, err = client.DoDigestAuth(ctx, req, res, sipgo.DigestAuth{
			Username: username,
			Password: password,
		})
		if err != nil {
			log.Error("Digest authentication failed",
				"error", err,
				"request_start_line", req.StartLine(),
			)
			return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
		}
		log.Info("Received status", "status", int(res.StatusCode))

		// Debug: Log successful digest auth
		log.Debug("Digest authentication completed successfully",
			"final_status", res.StatusCode,
			"final_reason", res.Reason,
		)
	}

	if res.StatusCode != 200 {
		return &RegisterResponseError{
			RegisterReq: req,
			RegisterRes: res,
			Msg:         res.StartLine(),
		}
	}

	// Now update server expiry
	t.expiry = t.opts.Expiry
	if h := res.GetHeader("Expires"); h != nil {
		val, err := strconv.Atoi(h.Value())
		if err != nil {
			return fmt.Errorf("Failed to parse server Expires value: %w", err)
		}
		t.expiry = time.Duration(val) * time.Second

		// Debug: Log server expiry update
		log.Debug("Server expiry updated",
			"client_expiry", t.opts.Expiry,
			"server_expiry", t.expiry,
			"expires_header_value", h.Value(),
		)
	}

	// Debug: Log successful registration
	log.Debug("Registration completed successfully",
		"final_expiry", t.expiry,
		"contact_uri", contact.Address.String(),
	)

	return nil
}

func (t *RegisterTransaction) QualifyLoop(ctx context.Context) error {
	// TODO: based on server response Expires header this must be adjusted
	// Allows caller to adjust

	calcRetry := func(expiry time.Duration) time.Duration {
		// Allow caller to use own interval
		if t.opts.RetryInterval != 0 {
			return t.opts.RetryInterval
		}

		calc := expiry.Seconds() * 0.75
		retry := time.Duration(calc) * time.Second

		// Set to 30 in case retry is not set
		if retry == 0 {
			retry = 30 * time.Second
		}

		return retry
	}

	expiry := t.expiry
	retry := calcRetry(expiry)

	t.log.Info("Starting registration refresh loop", "expiry", expiry, "retry_interval", retry)

	ticker := time.NewTicker(retry)
	defer ticker.Stop()

	// Wait for the first tick before sending any refresh
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
		// First tick received, now we can start the refresh loop
	}

	for {
		// Send refresh registration
		err := t.Qualify(ctx)
		if err != nil {
			t.log.Error("Registration refresh failed", "error", err)
			return err
		}

		// Check if expiry changed
		if t.expiry != expiry {
			// expiry got updated
			expiry = t.expiry
			retry = calcRetry(expiry)

			t.log.Info("Register expiry changed", "expiry_old", expiry, "expiry_new", t.expiry, "retry", retry)
			ticker.Reset(retry)
		}

		// Wait for next tick
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}

func (t *RegisterTransaction) Unregister(ctx context.Context) error {
	req := t.Origin

	req.RemoveHeader("Expires")
	req.RemoveHeader("Contact")
	req.AppendHeader(sip.NewHeader("Contact", "*"))
	expires := sip.ExpiresHeader(0)
	req.AppendHeader(&expires)
	return t.doRequest(ctx, req)
}

func (t *RegisterTransaction) Qualify(ctx context.Context) error {
	return t.doRequest(ctx, t.Origin)
}

func (t *RegisterTransaction) doRequest(ctx context.Context, req *sip.Request) error {
	// log := p.getLoggerCtx(ctx, "Register")
	log := t.log
	client := t.client
	username, password := t.opts.Username, t.opts.Password

	// Debug: Log connection preparation for subsequent requests
	log.Debug("Preparing to send subsequent REGISTER request",
		"client_name", client.Name(),
		"request_uri", req.Recipient.String(),
		"destination", req.Destination(),
		"proxy_host", t.opts.ProxyHost,
		"domain_vs_proxy", fmt.Sprintf("domain=%s, proxy=%s", req.Recipient.Host, t.opts.ProxyHost),
	)

	// Send request and parse response
	// req.SetDestination(*dst)
	req.RemoveHeader("Via")

	// Debug: Log before sending packet
	log.Debug("Sending subsequent REGISTER packet via client.Do",
		"request_start_line", req.StartLine(),
		"via_header_removed", true,
		"from_header", req.From(),
		"to_header", req.To(),
		"contact_header", req.Contact(),
		"expires_header", req.GetHeader("Expires"),
	)

	res, err := client.Do(ctx, req, sipgo.ClientRequestRegisterBuild)
	if err != nil {
		log.Error("Failed to send subsequent REGISTER packet",
			"error", err,
			"request_start_line", req.StartLine(),
		)
		return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
	}

	// Debug: Log successful packet sending
	log.Debug("Subsequent REGISTER packet sent successfully",
		"response_status", res.StatusCode,
		"response_reason", res.Reason,
		"response_via", res.Via(),
	)

	log.Info("Received status", "uri", req.Recipient.String())
	if res.StatusCode == sip.StatusUnauthorized || res.StatusCode == sip.StatusProxyAuthRequired {
		log.Info("Unathorized. Doing digest auth")

		// Debug: Log digest auth attempt for subsequent request
		log.Debug("Starting digest authentication for subsequent request",
			"username", username,
			"auth_header", res.GetHeader("WWW-Authenticate"),
		)

		res, err = client.DoDigestAuth(ctx, req, res, sipgo.DigestAuth{
			Username: username,
			Password: password,
		})
		if err != nil {
			log.Error("Digest authentication failed for subsequent request",
				"error", err,
				"request_start_line", req.StartLine(),
			)
			return fmt.Errorf("fail to get response req=%q : %w", req.StartLine(), err)
		}
		log.Info("Received status", "uri", req.Recipient.String())

		// Debug: Log successful digest auth for subsequent request
		log.Debug("Digest authentication completed successfully for subsequent request",
			"final_status", res.StatusCode,
			"final_reason", res.Reason,
		)
	}

	if res.StatusCode != 200 {
		return &RegisterResponseError{
			RegisterReq: req,
			RegisterRes: res,
			Msg:         res.StartLine(),
		}
	}

	// Check is expirese changed
	if h := res.GetHeader("Expires"); h != nil {
		val, err := strconv.Atoi(h.Value())
		if err != nil {
			return fmt.Errorf("Failed to parse server Expires value: %w", err)
		}
		oldExpiry := t.expiry
		t.expiry = time.Duration(val) * time.Second

		// Debug: Log server expiry update for subsequent request
		log.Debug("Server expiry updated for subsequent request",
			"old_expiry", oldExpiry,
			"new_expiry", t.expiry,
			"expires_header_value", h.Value(),
		)
	}

	// Debug: Log successful subsequent registration
	log.Debug("Subsequent registration completed successfully",
		"final_expiry", t.expiry,
		"contact_uri", req.Contact().Address.String(),
	)

	return nil
}

func getResponse(ctx context.Context, tx sip.ClientTransaction) (*sip.Response, error) {
	select {
	case <-tx.Done():
		return nil, fmt.Errorf("transaction died")
	case res := <-tx.Responses():
		return res, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
