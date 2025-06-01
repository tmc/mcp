# Changeman - Intelligent Change Management System

Changeman is an intelligent change management system that uses natural language processing to analyze change descriptions and automatically identify which tests might be affected by proposed changes.

## Features

- **Natural Language Analysis**: Parses change descriptions to understand the type and impact of changes
- **Test Discovery**: Automatically finds tests that might be affected by the change
- **Relevance Scoring**: Ranks tests by their likelihood of being affected
- **Change Classification**: Categorizes changes (bug fix, feature, refactor, etc.)
- **Impact Assessment**: Evaluates the potential impact level of changes

## Components

### Analyzer
The `Analyzer` component processes natural language descriptions to extract:
- Change type (bug fix, feature, refactor, etc.)
- Impact level (low, medium, high)
- Keywords and components mentioned
- Affected packages or modules

### TestFinder
The `TestFinder` component:
- Scans the codebase for test files
- Analyzes test functions and their dependencies
- Calculates relevance scores based on:
  - Package/component matches
  - Keyword matches in test names
  - Import dependencies
  - Change type considerations

### ChangeManager
The `ChangeManager` orchestrates the analysis:
- Coordinates between Analyzer and TestFinder
- Provides a unified interface for change analysis
- Generates comprehensive analysis reports

## Usage

### As a Library

```go
import "github.com/tmc/mcp/exp/changeman"

// Create a change manager
manager := changeman.NewChangeManager("/path/to/project")

// Analyze a change
analysis, err := manager.AnalyzeChange("Fix critical bug in authentication module")
if err != nil {
    log.Fatal(err)
}

// Print the analysis
fmt.Println(analysis.Summary())
```

### Command Line

```bash
# Build the tool
go build ./cmd/changeman

# Analyze a change
./changeman -root=/path/to/project -desc="Add new feature to support OAuth2 authentication"
```

## Example Output

```
Change Analysis:
  Type: Feature
  Impact: Medium
  Keywords: [add new feature support]
  Components: [OAuth2]

Affected Tests (5):
  auth.TestAuthenticate (relevance: 45%)
  auth.TestAuthorize (relevance: 40%)
  integration.TestOAuth2Flow (relevance: 85%)
  middleware.TestAuthMiddleware (relevance: 30%)
  handlers.TestLoginHandler (relevance: 25%)
```

## How It Works

1. **Change Analysis**: The system parses the natural language description to identify:
   - Action words (fix, add, refactor, etc.)
   - Technical terms and component names
   - Impact indicators (critical, major, minor, etc.)

2. **Test Discovery**: The system scans the codebase to find all test files and:
   - Parses Go test files to extract test functions
   - Analyzes package dependencies
   - Identifies naming patterns

3. **Relevance Calculation**: For each test, the system calculates a relevance score based on:
   - Direct component/package matches (highest weight)
   - Keyword matches in test names
   - Import dependencies
   - Change type (e.g., refactors affect more tests)

4. **Result Ranking**: Tests are ranked by relevance score, helping developers focus on the most likely affected tests first.

## Future Enhancements

- Machine learning for improved change classification
- Historical analysis to learn from past changes
- Integration with version control systems
- Support for multiple programming languages
- Test execution recommendations
- Change impact visualization