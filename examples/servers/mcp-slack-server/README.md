# MCP Slack Server

A Model Context Protocol server that provides tools for interacting with Slack workspaces.

## Features

- Send messages to channels
- List channels in workspace
- Get channel message history
- Create new channels
- Invite users to channels
- Get user information

## Tools

### send_message
Send a message to a Slack channel.
- `channel` (string, required): Channel ID or name to send message to
- `text` (string, required): Message text to send
- `thread_ts` (string, optional): Timestamp of thread to reply to

### list_channels
List all channels in the workspace.
- `types` (string, optional): Comma-separated list of channel types (default: public_channel,private_channel)
- `limit` (integer, optional): Maximum number of channels to return (default: 100)

### get_channel_history
Get message history from a channel.
- `channel` (string, required): Channel ID to get history from
- `limit` (integer, optional): Number of messages to retrieve (default: 10)
- `oldest` (string, optional): Oldest timestamp of messages to include

### create_channel
Create a new channel.
- `name` (string, required): Name of the channel to create
- `is_private` (boolean, optional): Whether the channel should be private (default: false)

### invite_to_channel
Invite users to a channel.
- `channel` (string, required): Channel ID to invite users to
- `users` (string, required): Comma-separated list of user IDs

### get_user_info
Get information about a user.
- `user` (string, required): User ID to get information about

## Usage

```bash
go run main.go
```

## Building

```bash
go build -o mcp-slack-server .
```