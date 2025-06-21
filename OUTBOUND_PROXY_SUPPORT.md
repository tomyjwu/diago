# Outbound Proxy Support for INVITE Calls

## Overview

This document describes the implementation of outbound proxy support for INVITE calls in the Diago SIP library, similar to the existing support for REGISTER requests.

## Problem Statement

When using UUID domains for SIP addressing (e.g., `sip:callee@550e8400-e29b-41d4-a716-446655440000`), the actual SIP server might be located at a different address (e.g., `proxy.example.com:5060`). The current implementation only supported outbound proxies for REGISTER requests, but not for INVITE calls.

## Solution

The implementation adds outbound proxy support for INVITE calls by:

1. Adding a `ProxyHost` field to `InviteOptions` and `InviteClientOptions` structs
2. Using `SetDestination()` to route INVITE requests through the specified proxy
3. Preserving the original domain in the SIP headers while routing the actual connection through the proxy

## Changes Made

### 1. Updated InviteOptions struct (diago.go)
```go
type InviteOptions struct {
    // ... existing fields ...
    // Outbound proxy host:port for routing INVITE requests
    ProxyHost string
}
```

### 2. Updated InviteClientOptions struct (dialog_client_session.go)
```go
type InviteClientOptions struct {
    // ... existing fields ...
    // Outbound proxy host:port for routing INVITE requests
    ProxyHost string
}
```

### 3. Added proxy routing in Invite method (dialog_client_session.go)
```go
// Set outbound proxy if specified
if opts.ProxyHost != "" {
    inviteReq.SetDestination(opts.ProxyHost)
    // Debug logging
    slog.Default().Info("INVITE using outbound proxy",
        "recipient_uri", inviteReq.Recipient.String(),
        "proxy_host", opts.ProxyHost,
        "domain_vs_proxy", fmt.Sprintf("domain=%s, proxy=%s", inviteReq.Recipient.Host, opts.ProxyHost),
    )
}
```

### 4. Updated Invite and InviteBridge methods to pass ProxyHost
Both methods now pass the `ProxyHost` from `InviteOptions` to `InviteClientOptions`.

## Usage Example

```go
// Create INVITE with UUID domain and outbound proxy
callRecipient := sip.Uri{}
sip.ParseUri("sip:callee@550e8400-e29b-41d4-a716-446655440000", &callRecipient)

// Create dialog session with outbound proxy
dialog, err := dg.Invite(ctx, callRecipient, InviteOptions{
    ProxyHost: "proxy.example.com:5060",
})
if err != nil {
    return err
}

// The INVITE will be sent to proxy.example.com:5060
// but the To header will still contain the UUID domain
// This allows the proxy to route based on the domain while
// the actual connection goes through the proxy
```

## How It Works

1. **Domain Preservation**: The original domain (UUID) is preserved in the SIP headers (To, From, etc.)
2. **Proxy Routing**: The actual network connection is made to the specified proxy
3. **Header Construction**: The underlying sipgo library constructs proper Via and To headers
4. **Debug Logging**: Added logging to show the distinction between domain and proxy

## Testing

A test case `TestInviteWithOutboundProxy` verifies that:
- The request destination is set to the proxy
- The To header still contains the original domain
- Debug logging shows the correct domain vs proxy configuration

## Benefits

1. **Consistent API**: Same pattern as REGISTER outbound proxy support
2. **UUID Domain Support**: Enables proper routing when using UUID domains
3. **Flexible Routing**: Allows different proxies for different types of requests
4. **Debug Visibility**: Clear logging shows how requests are routed

## Compatibility

This change is backward compatible - existing code without `ProxyHost` will continue to work as before, sending requests directly to the recipient URI's host.

## Connection Reuse Between REGISTER and INVITE

### How Connection Reuse Works

The current implementation **already supports TCP connection reuse** between REGISTER and INVITE requests when using the same transport and destination:

1. **Same Client Instance**: Both REGISTER and INVITE use the same `sipgo.Client` instance for a given transport
2. **Underlying Connection Pooling**: The `sipgo` library manages connection pooling at the transport layer
3. **Automatic Reuse**: When using the same outbound proxy for both REGISTER and INVITE, the same TCP connection is reused

### Connection Reuse Scenarios

#### ✅ **Connection Reused** (Same Transport + Same Proxy)
```go
// Both use the same transport and proxy
regOpts := RegisterOptions{
    ProxyHost: "proxy.example.com:5060",
}

inviteOpts := InviteOptions{
    ProxyHost: "proxy.example.com:5060",
}

// Result: Same TCP connection reused
```

#### ❌ **Different Connections** (Different Destinations)
```go
// Different proxies = different connections
regOpts := RegisterOptions{
    ProxyHost: "proxy1.example.com:5060",
}

inviteOpts := InviteOptions{
    ProxyHost: "proxy2.example.com:5060",
}

// Result: Different TCP connections
```

#### ❌ **Different Connections** (Different Transports)
```go
// Different transport types = different connections
regOpts := RegisterOptions{
    ProxyHost: "proxy.example.com:5060", // Uses UDP transport
}

inviteOpts := InviteOptions{
    ProxyHost: "proxy.example.com:5060", // Uses TCP transport
}

// Result: Different connections (UDP vs TCP)
```

### Best Practices for Connection Reuse

1. **Use Same Transport**: Ensure both REGISTER and INVITE use the same transport type (TCP, UDP, etc.)
2. **Use Same Proxy**: Use the same `ProxyHost` for both REGISTER and INVITE when possible
3. **Keep Connections Alive**: The underlying `sipgo` library handles connection lifecycle management

### Future Enhancements

The implementation includes a placeholder for future proxy-specific client caching:

```go
// getClientForProxy returns a client optimized for a specific proxy destination
func (dg *Diago) getClientForProxy(proxyHost string, tran *Transport) *sipgo.Client {
    // TODO: Implement proxy-specific client caching for better connection reuse
}
```

This could be enhanced to maintain a cache of clients per proxy destination for optimal connection reuse. 