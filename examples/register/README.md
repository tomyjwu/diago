# Register Example with Debug Logging

This example demonstrates SIP registration with comprehensive debug logging to trace the packet sending flow.

## Features

- **Packet Preparation Tracing**: See how SIP REGISTER packets are prepared with headers and parameters
- **Connection Setup Tracing**: Monitor how connections are configured for different domain vs proxy scenarios
- **Packet Sending Tracing**: Track the actual sending of packets through the SIP client
- **NAT Traversal Support**: Debug logging for NAT traversal scenarios with rport/received parameters
- **Digest Authentication**: Detailed logging of authentication flows

## Usage

### Basic Registration
```bash
go run . -username alice -password secret sip:alice@example.com
```

### Registration with Outbound Proxy
```bash
go run . -username alice -password secret -proxy 192.168.1.100:5060 sip:alice@example.com
```

### Registration with Debug Logging
```bash
go run . -username alice -password secret -proxy 192.168.1.100:5060 -debug sip:alice@example.com
```

## Debug Logging Output

When using the `-debug` flag, you'll see detailed logs showing:

### 1. Packet Preparation
```
DEBUG Register packet prepared recipient_uri=sip:alice@example.com contact_uri=sip:diago-register@192.168.1.50:15060 proxy_host=192.168.1.100:5060 expiry_seconds=3600 allow_headers=[] username=alice request_method=REGISTER request_uri=sip:alice@example.com destination=192.168.1.100:5060
```

### 2. Connection Preparation
```
DEBUG Selecting transport for registration recipient_uri=sip:alice@example.com transport_from_uri= selected_transport=udp proxy_host=192.168.1.100:5060
DEBUG Transport selected for registration transport_id= transport_protocol=udp bind_host=127.0.0.1 bind_port=15060 external_host=192.168.1.50 external_port=15060 tls_enabled=false
DEBUG Contact header configured for registration contact_uri=sip:diago-register@192.168.1.50:15060 contact_scheme=sip contact_user=diago-register contact_host=192.168.1.50 contact_port=15060 external_host_used=192.168.1.50 external_port_used=15060
```

### 3. Packet Sending
```
DEBUG Preparing to send REGISTER request client_name=diago-register request_uri=sip:alice@example.com destination=192.168.1.100:5060 contact_uri=sip:diago-register@192.168.1.50:15060 proxy_host=192.168.1.100:5060 domain_vs_proxy=domain=example.com, proxy=192.168.1.100:5060
DEBUG Sending REGISTER packet via client.Do request_start_line="REGISTER sip:alice@example.com SIP/2.0" via_header= from_header= to_header= contact_header= expires_header= allow_header=
```

### 4. NAT Traversal (if applicable)
```
DEBUG NAT traversal updates applied rport=15060 received=192.168.1.50 updated_contact=sip:diago-register@192.168.1.50:15060
```

### 5. Authentication (if required)
```
DEBUG Starting digest authentication username=alice auth_header=WWW-Authenticate: Digest realm="example.com", nonce="..."
DEBUG Digest authentication completed successfully final_status=200 final_reason=OK
```

## Key Debug Information

### Domain vs Proxy Configuration
- **Domain**: Used in the SIP URI (e.g., `sip:alice@example.com`)
- **Proxy**: Used for actual connection (e.g., `192.168.1.100:5060`)
- The debug logs show both values to help understand the routing

### Transport Configuration
- **Bind Host/Port**: Local interface for listening
- **External Host/Port**: Public address for Contact header
- **TLS**: Whether secure transport is used

### Packet Flow
1. **Preparation**: Headers are added to the request
2. **Connection**: Client is configured with transport settings
3. **Sending**: Packet is sent via `client.Do()`
4. **Response**: Response is processed and logged
5. **NAT Updates**: Contact header updated if NAT traversal detected
6. **Authentication**: Digest auth if required

## Environment Variables

You can also use environment variables for debug logging:

```bash
export LOG_LEVEL=DEBUG
export SIP_DEBUG=true
go run . -username alice -password secret sip:alice@example.com
```

## Troubleshooting

### Common Issues

1. **Connection Refused**: Check if the proxy server is reachable
2. **Authentication Failed**: Verify username/password
3. **NAT Issues**: Ensure external host/port are correctly configured
4. **Transport Issues**: Verify the transport protocol (UDP/TCP/TLS)

### Debug Tips

- Use `-debug` flag to see detailed packet flow
- Check the `domain_vs_proxy` field to verify routing
- Monitor `contact_uri` to ensure correct Contact header
- Look for NAT traversal updates in the logs

Run server and see how it registers
```bash
go run ./examples/register -username <username> -password <pass> sip:myuser@127.0.0.1:5060 
```

