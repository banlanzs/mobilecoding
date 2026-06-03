# MobileCoding - Remote Control for AI CLI

This skill enables remote control of Claude Code from your mobile phone.

## When to use

Use this skill when the user wants to:
- Control Claude Code from their mobile phone
- Start a remote control session
- Connect mobile device to the current CLI session

## How to use

When the user types `/mobilecoding` or asks to start remote control:

1. **Start the relay connection** by running:
   ```bash
   mobilecoding-relay --relay ws://localhost:8443/relay/agent
   ```

2. **Display the pairing information** to the user:
   - The relay CLI will output a session ID and pairing secret
   - Tell the user to scan the QR code or enter the pairing info on their phone

3. **Keep the connection alive** - the relay CLI will run until the user stops it

## Example interaction

User: `/mobilecoding`

You should:
1. Run `mobilecoding-relay` in the background
2. Show the pairing information
3. Explain how to connect from the phone

## Notes

- The relay server must be running (starts with `mobilecoding` command)
- The phone and computer must be on the same network
- Default relay URL is `ws://localhost:8443`
- The connection is bidirectional - phone can send commands and see output

## Troubleshooting

If connection fails:
1. Check if relay server is running: `mobilecoding`
2. Verify network connectivity
3. Check firewall settings
4. Try using the computer's IP address instead of localhost
