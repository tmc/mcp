package changemanagement

import (
	"fmt"
	"strings"
	"time"
)

// Documentation represents a generated documentation file
type Documentation struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Type     string `json:"type"`
}

// DocumentationGenerator generates documentation for changes
type DocumentationGenerator struct {
}

// NewDocumentationGenerator creates a new documentation generator
func NewDocumentationGenerator() *DocumentationGenerator {
	return &DocumentationGenerator{}
}

// GenerateDocs generates documentation for a change
func (d *DocumentationGenerator) GenerateDocs(analysis *AnalysisResult, format string) ([]Documentation, error) {
	docs := []Documentation{}

	// Generate overview documentation
	overview := d.generateOverview(analysis, format)
	docs = append(docs, Documentation{
		Filename: fmt.Sprintf("CHANGE_%s.%s", analysis.Type, getExtension(format)),
		Content:  overview,
		Type:     "overview",
	})

	// Generate migration guide if needed
	if analysis.Breaking || analysis.Type == ChangeTypeMigration {
		migration := d.generateMigrationGuide(analysis, format)
		docs = append(docs, Documentation{
			Filename: fmt.Sprintf("MIGRATION_GUIDE.%s", getExtension(format)),
			Content:  migration,
			Type:     "migration",
		})
	}

	// Generate API documentation for API changes
	if analysis.Category == "api" {
		apiDoc := d.generateAPIDoc(analysis, format)
		docs = append(docs, Documentation{
			Filename: fmt.Sprintf("API_CHANGES.%s", getExtension(format)),
			Content:  apiDoc,
			Type:     "api",
		})
	}

	// Generate security documentation for security changes
	if analysis.Type == ChangeTypeSecurity || analysis.Category == "authentication" {
		securityDoc := d.generateSecurityDoc(analysis, format)
		docs = append(docs, Documentation{
			Filename: fmt.Sprintf("SECURITY_NOTES.%s", getExtension(format)),
			Content:  securityDoc,
			Type:     "security",
		})
	}

	// Generate release notes
	releaseNotes := d.generateReleaseNotes(analysis, format)
	docs = append(docs, Documentation{
		Filename: fmt.Sprintf("RELEASE_NOTES.%s", getExtension(format)),
		Content:  releaseNotes,
		Type:     "release",
	})

	return docs, nil
}

func (d *DocumentationGenerator) generateOverview(analysis *AnalysisResult, format string) string {
	var sb strings.Builder

	if format == "markdown" {
		sb.WriteString(fmt.Sprintf("# %s Change Overview\n\n", strings.Title(string(analysis.Type))))
		sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
		
		sb.WriteString("## Summary\n\n")
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", analysis.Type))
		sb.WriteString(fmt.Sprintf("- **Category**: %s\n", analysis.Category))
		sb.WriteString(fmt.Sprintf("- **Risk Level**: %s\n", analysis.RiskLevel))
		sb.WriteString(fmt.Sprintf("- **Breaking Change**: %v\n\n", analysis.Breaking))
		
		if len(analysis.Components) > 0 {
			sb.WriteString("## Affected Components\n\n")
			for _, comp := range analysis.Components {
				sb.WriteString(fmt.Sprintf("- %s\n", comp))
			}
			sb.WriteString("\n")
		}
		
		if len(analysis.Requirements.Functional) > 0 {
			sb.WriteString("## Functional Requirements\n\n")
			for _, req := range analysis.Requirements.Functional {
				sb.WriteString(fmt.Sprintf("- %s\n", req))
			}
			sb.WriteString("\n")
		}
		
		if len(analysis.Recommendations) > 0 {
			sb.WriteString("## Recommendations\n\n")
			for _, rec := range analysis.Recommendations {
				sb.WriteString(fmt.Sprintf("### %s\n", rec.Type))
				sb.WriteString(fmt.Sprintf("%s (confidence: %.2f)\n\n", rec.Suggestion, rec.Confidence))
			}
		}
	}

	return sb.String()
}

func (d *DocumentationGenerator) generateMigrationGuide(analysis *AnalysisResult, format string) string {
	var sb strings.Builder

	if format == "markdown" {
		sb.WriteString("# Migration Guide\n\n")
		sb.WriteString(fmt.Sprintf("**Change Type**: %s\n", analysis.Type))
		sb.WriteString(fmt.Sprintf("**Risk Level**: %s\n\n", analysis.RiskLevel))
		
		sb.WriteString("## Overview\n\n")
		sb.WriteString("This guide helps you migrate to the new implementation.\n\n")
		
		sb.WriteString("## Migration Steps\n\n")
		sb.WriteString("1. **Backup current state**\n")
		sb.WriteString("   ```bash\n")
		sb.WriteString("   # Backup your data\n")
		sb.WriteString("   backup-tool create\n")
		sb.WriteString("   ```\n\n")
		
		sb.WriteString("2. **Update dependencies**\n")
		sb.WriteString("   ```bash\n")
		sb.WriteString("   # Update package versions\n")
		sb.WriteString("   package-manager update\n")
		sb.WriteString("   ```\n\n")
		
		sb.WriteString("3. **Apply changes**\n")
		sb.WriteString("   - Update configuration files\n")
		sb.WriteString("   - Modify code as needed\n")
		sb.WriteString("   - Run migration scripts\n\n")
		
		sb.WriteString("4. **Test thoroughly**\n")
		sb.WriteString("   ```bash\n")
		sb.WriteString("   # Run test suite\n")
		sb.WriteString("   test-runner --all\n")
		sb.WriteString("   ```\n\n")
		
		if analysis.Breaking {
			sb.WriteString("## Breaking Changes\n\n")
			sb.WriteString("⚠️ **Warning**: This migration includes breaking changes.\n\n")
			sb.WriteString("### Affected Areas\n\n")
			for _, area := range analysis.AffectedAreas {
				sb.WriteString(fmt.Sprintf("- %s\n", area))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (d *DocumentationGenerator) generateAPIDoc(analysis *AnalysisResult, format string) string {
	var sb strings.Builder

	if format == "markdown" {
		sb.WriteString("# API Changes Documentation\n\n")
		sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02")))
		
		sb.WriteString("## Overview\n\n")
		sb.WriteString("This document describes the API changes introduced by this update.\n\n")
		
		sb.WriteString("## Changed Endpoints\n\n")
		sb.WriteString("### Before\n\n")
		sb.WriteString("```http\n")
		sb.WriteString("GET /api/v1/endpoint\n")
		sb.WriteString("Authorization: Basic <credentials>\n")
		sb.WriteString("```\n\n")
		
		sb.WriteString("### After\n\n")
		sb.WriteString("```http\n")
		sb.WriteString("GET /api/v2/endpoint\n")
		sb.WriteString("Authorization: Bearer <token>\n")
		sb.WriteString("```\n\n")
		
		sb.WriteString("## New Features\n\n")
		for _, req := range analysis.Requirements.Functional {
			sb.WriteString(fmt.Sprintf("- %s\n", req))
		}
		sb.WriteString("\n")
		
		sb.WriteString("## Migration Notes\n\n")
		sb.WriteString("- Update client libraries to latest version\n")
		sb.WriteString("- Review authentication flow changes\n")
		sb.WriteString("- Test all integrations thoroughly\n")
	}

	return sb.String()
}

func (d *DocumentationGenerator) generateSecurityDoc(analysis *AnalysisResult, format string) string {
	var sb strings.Builder

	if format == "markdown" {
		sb.WriteString("# Security Notes\n\n")
		sb.WriteString("## Security Improvements\n\n")
		
		if analysis.Category == "authentication" {
			sb.WriteString("### Authentication Changes\n\n")
			sb.WriteString("- Enhanced authentication mechanism\n")
			sb.WriteString("- Improved token management\n")
			sb.WriteString("- Added multi-factor authentication support\n\n")
		}
		
		sb.WriteString("## Security Considerations\n\n")
		sb.WriteString("1. **Review access controls**\n")
		sb.WriteString("2. **Update security policies**\n")
		sb.WriteString("3. **Conduct security audit**\n")
		sb.WriteString("4. **Monitor for suspicious activity**\n\n")
		
		sb.WriteString("## Compliance\n\n")
		sb.WriteString("This change maintains compliance with:\n")
		sb.WriteString("- OWASP security standards\n")
		sb.WriteString("- Industry best practices\n")
		sb.WriteString("- Regulatory requirements\n")
	}

	return sb.String()
}

func (d *DocumentationGenerator) generateReleaseNotes(analysis *AnalysisResult, format string) string {
	var sb strings.Builder

	if format == "markdown" {
		sb.WriteString("# Release Notes\n\n")
		sb.WriteString(fmt.Sprintf("**Date**: %s\n", time.Now().Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("**Change Type**: %s\n\n", analysis.Type))
		
		sb.WriteString("## What's Changed\n\n")
		
		// Format based on change type
		switch analysis.Type {
		case ChangeTypeFeature:
			sb.WriteString("### New Features\n\n")
		case ChangeTypeBugFix:
			sb.WriteString("### Bug Fixes\n\n")
		case ChangeTypePerformance:
			sb.WriteString("### Performance Improvements\n\n")
		case ChangeTypeSecurity:
			sb.WriteString("### Security Updates\n\n")
		}
		
		for _, req := range analysis.Requirements.Functional {
			sb.WriteString(fmt.Sprintf("- %s\n", req))
		}
		sb.WriteString("\n")
		
		if analysis.Breaking {
			sb.WriteString("### Breaking Changes\n\n")
			sb.WriteString("⚠️ This release contains breaking changes. Please review the migration guide.\n\n")
		}
		
		sb.WriteString("## Upgrade Instructions\n\n")
		sb.WriteString("1. Review the migration guide\n")
		sb.WriteString("2. Update dependencies\n")
		sb.WriteString("3. Run tests\n")
		sb.WriteString("4. Deploy with monitoring\n")
	}

	return sb.String()
}

func getExtension(format string) string {
	switch format {
	case "markdown":
		return "md"
	case "html":
		return "html"
	default:
		return "txt"
	}
}