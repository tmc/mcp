# mcpdiff TODO

## Single File Shadow Support

The mcpdiff tool should support comparing primary and shadow records within a single trace file, without requiring a `-compare` flag.

### Current Behavior
- mcpdiff currently requires exactly 2 files
- Tests expect a `-compare` flag that doesn't exist

### Expected Behavior  
When given a single trace file:
1. Automatically detect if the file contains shadow records (lines with `mcp-send-shadow` or `mcp-recv-shadow`)
2. Split the records into primary and shadow groups
3. Compare primary vs shadow records without needing a special flag
4. Match shadow records to their primary counterparts using span IDs or the `linksto` attribute

### Implementation Notes
- Shadow records have a direction like `send-shadow` or `recv-shadow`
- Shadow records typically have a `linksto` attribute pointing to the primary record's span ID
- A trace file with `compare=true` in the header indicates it contains shadow data
- Records should be matched by their relationship (shadow linking to primary)

### Example
```
mcp-send {"jsonrpc":"2.0","method":"test","id":1} # 1000.000 spanid=req1
mcp-send {"jsonrpc":"2.0","result":"ok","id":1} # 1001.000 spanid=resp1  
mcp-send-shadow {"jsonrpc":"2.0","result":"ok","id":1} # 1001.100 spanid=shadow1 linksto=resp1
```

In this case, mcpdiff should automatically compare the `mcp-send` response with the `mcp-send-shadow` response.

### Test Fixes Needed
Remove references to the non-existent `-compare` flag in:
- `testdata/compare-mode.txt`
- `shadow_test.go`

Update mcpdiff to handle single file input when it contains shadow records.