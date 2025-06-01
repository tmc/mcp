# Renaming to jsonschema2gostruct

If you want to rename this tool from `jsonrpc2gostruct` to `jsonschema2gostruct`, here are the steps to follow:

1. Create the new directory:
```bash
mkdir -p /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonschema2gostruct
```

2. Copy all files to the new directory:
```bash
cp -r /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonrpc2gostruct/* /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonschema2gostruct/
```

3. Rename references in README.md:
```bash
sed -i '' 's/jsonrpc2gostruct/jsonschema2gostruct/g' /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonschema2gostruct/README.md
```

4. Update package comment in main.go:
```bash
sed -i '' '1s/^/\/\/ Package main implements jsonschema2gostruct, a tool to convert JSON Schema to Go structs.\n/' /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonschema2gostruct/main.go
```

5. Build and test the new tool:
```bash
cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonschema2gostruct
go build
./jsonschema2gostruct -h
```

6. Remove the old directory if everything looks good (optional):
```bash
rm -rf /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/jsonrpc2gostruct
```

The rename primarily reflects the tool's focus on converting JSON Schema to Go structs rather than being limited to JSON-RPC messages.