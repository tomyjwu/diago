# Debug Logging Enhancements for Register Packet Flow

This document summarizes the comprehensive debug logging added to trace the SIP REGISTER packet sending flow in the diago library.

## Overview

The debug logging has been enhanced to provide detailed visibility into:
1. **Packet Preparation**: How SIP REGISTER packets are constructed
2. **Connection Setup**: How connections are configured for domain vs proxy scenarios
3. **Packet Sending**: The actual sending of packets through the SIP client
4. **NAT Traversal**: Updates to Contact headers for NAT scenarios
5. **Authentication**: Digest authentication flows

## Files Modified

### 1. `register_transaction.go`

#### `newRegisterTransaction()` function
- Added debug logging for packet preparation details
- Logs recipient URI, contact URI, proxy host, expiry, headers, and destination

#### `Register()` function
- Added connection preparation logging showing domain vs proxy configuration
- Added packet sending logging before and after `client.Do()` calls
- Added NAT traversal logging for rport/received parameter updates
- Added digest authentication logging for unauthorized responses
- Added server expiry update logging

#### `doRequest()` function
- Added similar debug logging for subsequent registration requests
- Logs connection preparation, packet sending, and authentication flows

### 2. `diago.go`

#### `createClient()` function
- Added debug logging for SIP client creation
- Logs transport configuration, bind settings, and client options

#### `RegisterTransaction()` function
- Added transport selection logging
- Added transport details logging
- Added contact header configuration logging
- Added client selection logging

#### `contactHDRFromTransport()` function
- Added debug logging for contact URI construction
- Shows how external host/port are used in Contact headers

### 3. `examples/register/main.go`

#### Enhanced example application
- Added `-proxy` flag for outbound proxy configuration
- Added `-debug` flag to enable debug logging
- Added comprehensive usage examples
- Added registration configuration logging

#### `examples/register/README.md`
- Created comprehensive documentation
- Shows example debug output
- Explains key debug information fields
- Provides troubleshooting guidance

## Debug Log Categories

### 1. Packet Preparation
```
DEBUG Register packet prepared
- recipient_uri: The target SIP URI
- contact_uri: The Contact header URI
- proxy_host: Outbound proxy if configured
- expiry_seconds: Registration expiry time
- request_method: REGISTER
- request_uri: The request URI
- destination: Where the packet will be sent
```

### 2. Connection Preparation
```
DEBUG Preparing to send REGISTER request
- client_name: Name of the SIP client
- request_uri: The target URI
- destination: Connection destination
- contact_uri: Contact header URI
- proxy_host: Outbound proxy
- domain_vs_proxy: Comparison of domain vs proxy
```

### 3. Transport Configuration
```
DEBUG Transport selected for registration
- transport_id: Transport identifier
- transport_protocol: UDP/TCP/TLS
- bind_host: Local bind address
- bind_port: Local bind port
- external_host: Public address for Contact
- external_port: Public port for Contact
- tls_enabled: Whether TLS is used
```

### 4. Packet Sending
```
DEBUG Sending REGISTER packet via client.Do
- request_start_line: Full request start line
- via_header: Via header details
- from_header: From header details
- to_header: To header details
- contact_header: Contact header details
- expires_header: Expires header value
- allow_header: Allow header value
```

### 5. NAT Traversal
```
DEBUG NAT traversal updates applied
- rport: Port from rport parameter
- received: IP from received parameter
- updated_contact: Updated Contact URI
```

### 6. Authentication
```
DEBUG Starting digest authentication
- username: Authentication username
- auth_header: WWW-Authenticate header
```

## Key Debug Information Fields

### Domain vs Proxy Configuration
The debug logs clearly show the distinction between:
- **Domain**: Used in the SIP URI (e.g., `sip:alice@example.com`)
- **Proxy**: Used for actual connection (e.g., `192.168.1.100:5060`)

This is crucial for understanding routing when domain and outbound proxy are different.

### Transport Configuration
Shows how the transport layer is configured:
- **Bind settings**: Local interface for listening
- **External settings**: Public address for Contact headers
- **TLS configuration**: Whether secure transport is used

### Contact Header Construction
Tracks how the Contact header is built from transport configuration:
- **Scheme**: sip/sips based on TLS
- **User**: User agent name
- **Host**: External host from transport
- **Port**: External port from transport

## Usage Examples

### Basic Registration with Debug
```bash
go run examples/register/main.go -username alice -password secret -debug sip:alice@example.com
```

### Registration with Proxy and Debug
```bash
go run examples/register/main.go -username alice -password secret -proxy 192.168.1.100:5060 -debug sip:alice@example.com
```

### Environment Variable Debug
```bash
export LOG_LEVEL=DEBUG
export SIP_DEBUG=true
go run examples/register/main.go -username alice -password secret sip:alice@example.com
```

## Benefits

1. **Packet Flow Visibility**: Complete trace of packet preparation and sending
2. **Connection Debugging**: Clear view of how connections are established
3. **NAT Troubleshooting**: Visibility into NAT traversal updates
4. **Authentication Debugging**: Detailed auth flow logging
5. **Proxy Configuration**: Clear distinction between domain and proxy routing
6. **Transport Debugging**: Complete transport layer configuration visibility

## Troubleshooting Scenarios

### Connection Issues
- Check `destination` field to verify where packets are sent
- Verify `proxy_host` configuration
- Check transport bind settings

### NAT Issues
- Monitor `external_host` and `external_port` in Contact headers
- Look for NAT traversal updates in logs
- Verify rport/received parameter handling

### Authentication Issues
- Check digest auth flow logs
- Verify username/password configuration
- Monitor WWW-Authenticate header processing

### Transport Issues
- Verify transport protocol selection
- Check TLS configuration
- Monitor bind port availability

This comprehensive debug logging provides complete visibility into the SIP registration packet flow, making it much easier to diagnose and troubleshoot registration issues. 