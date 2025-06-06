package sourcereflect

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// CallerInfo contains information about the calling function
type CallerInfo struct {
	Function string
	File     string
	Line     int
	Package  string
}

// GetCallerInfo returns information about the calling function
func GetCallerInfo(skip int) (*CallerInfo, error) {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return nil, fmt.Errorf("failed to get caller info")
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return nil, fmt.Errorf("failed to get function info")
	}

	return &CallerInfo{
		Function: fn.Name(),
		File:     file,
		Line:     line,
		Package:  getPackageFromFunc(fn.Name()),
	}, nil
}

func getPackageFromFunc(funcName string) string {
	// Function names are formatted as "package.Function" or "package.Type.Method"
	lastSlash := strings.LastIndex(funcName, "/")
	if lastSlash < 0 {
		lastSlash = 0
	}

	firstDot := strings.Index(funcName[lastSlash:], ".")
	if firstDot < 0 {
		return ""
	}

	return funcName[:lastSlash+firstDot]
}

// SchemaFromCaller generates a JSON schema from a type in the calling context
func SchemaFromCaller(v interface{}) (*Schema, error) {
	callerInfo, err := GetCallerInfo(1)
	if err != nil {
		return nil, fmt.Errorf("failed to get caller info: %w", err)
	}

	schema, err := FromValue(v)
	if err != nil {
		return nil, err
	}

	// Add caller metadata to the schema
	if schema.Additional == nil {
		schema.Additional = make(map[string]interface{})
	}
	schema.Additional["$sourceLocation"] = map[string]interface{}{
		"file":     callerInfo.File,
		"line":     callerInfo.Line,
		"function": callerInfo.Function,
		"package":  callerInfo.Package,
	}

	return schema, nil
}

// TypeSchemaFromCaller generates a JSON schema from a type using caller context
func TypeSchemaFromCaller(t reflect.Type) (*Schema, error) {
	callerInfo, err := GetCallerInfo(1)
	if err != nil {
		return nil, fmt.Errorf("failed to get caller info: %w", err)
	}

	schema, err := FromType(t)
	if err != nil {
		return nil, err
	}

	// Add caller metadata to the schema
	if schema.Additional == nil {
		schema.Additional = make(map[string]interface{})
	}
	schema.Additional["$sourceLocation"] = map[string]interface{}{
		"file":     callerInfo.File,
		"line":     callerInfo.Line,
		"function": callerInfo.Function,
		"package":  callerInfo.Package,
	}

	return schema, nil
}
