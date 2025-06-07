# MCP AWS Knowledge Base Server

A Model Context Protocol (MCP) server that provides access to AWS Knowledge Bases through the Bedrock Agent Runtime service.

## Features

### Tools

- **query_knowledge_base** - Query an AWS Knowledge Base using Bedrock Agent Runtime
  - Retrieve relevant documents from a knowledge base
  - Support for result filtering and customizable result counts
  - Returns scored results with source locations

- **retrieve_and_generate** - Retrieve from knowledge base and generate a response
  - Combines retrieval with AI generation using the retrieved context
  - Supports custom model ARNs
  - Includes citation information in responses

## Configuration

### Required Environment Variables

- `AWS_REGION` - AWS region where your knowledge base is located
- `AWS_ACCESS_KEY_ID` - Your AWS access key ID (or use IAM roles)
- `AWS_SECRET_ACCESS_KEY` - Your AWS secret access key (or use IAM roles)

### Optional Environment Variables

- `AWS_PROFILE` - AWS profile to use (alternative to access keys)
- `AWS_ROLE_ARN` - ARN of IAM role to assume

### Required AWS Permissions

The credentials/role must have the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "bedrock:Retrieve",
                "bedrock:RetrieveAndGenerate"
            ],
            "Resource": [
                "arn:aws:bedrock:*:*:knowledge-base/*"
            ]
        }
    ]
}
```

## Usage

### Starting the Server

```bash
go run .
```

### Example Tool Calls

#### Query Knowledge Base

```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "query_knowledge_base",
        "arguments": {
            "knowledge_base_id": "your-kb-id-here",
            "query": "What are the company policies?",
            "max_results": 10
        }
    }
}
```

#### Retrieve and Generate

```json
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call", 
    "params": {
        "name": "retrieve_and_generate",
        "arguments": {
            "knowledge_base_id": "your-kb-id-here",
            "input_text": "Explain the company vacation policy",
            "model_arn": "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-sonnet-20240229-v1:0"
        }
    }
}
```

## Testing

Run the test suite:

```bash
go test -v .
```

### Test Coverage

- Basic server functionality and initialization
- Tool registration and availability
- Error handling for invalid requests
- Performance testing with aggressive timeouts
- AWS API integration (mocked for testing)

## Dependencies

- **AWS SDK for Go v2** - Official AWS SDK for Go
- **Bedrock Agent Runtime** - AWS service for knowledge base operations
- **MCP SDK** - Model Context Protocol implementation

## Error Handling

The server handles various error conditions:

- Missing or invalid AWS credentials
- Invalid knowledge base IDs
- Network timeouts and AWS service errors
- Malformed tool arguments
- JSON-RPC protocol errors

## Performance

- Supports aggressive timeout testing (25ms-100ms)
- Handles concurrent requests
- Efficient memory usage with streaming responses
- Connection pooling via AWS SDK

## Security

- Uses AWS IAM for authentication and authorization
- No credentials stored in code
- Respects AWS service limits and quotas
- Input validation for all tool parameters