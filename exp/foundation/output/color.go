package output

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ColorProvider provides color formatting for output.
type ColorProvider struct {
	enabled bool
	scheme  ColorScheme
}

// Color codes for ANSI terminal colors.
const (
	colorReset = "\033[0m"

	// Regular colors
	colorBlack   = "\033[0;30m"
	colorRed     = "\033[0;31m"
	colorGreen   = "\033[0;32m"
	colorYellow  = "\033[0;33m"
	colorBlue    = "\033[0;34m"
	colorMagenta = "\033[0;35m"
	colorCyan    = "\033[0;36m"
	colorWhite   = "\033[0;37m"

	// Bold colors
	colorBoldBlack   = "\033[1;30m"
	colorBoldRed     = "\033[1;31m"
	colorBoldGreen   = "\033[1;32m"
	colorBoldYellow  = "\033[1;33m"
	colorBoldBlue    = "\033[1;34m"
	colorBoldMagenta = "\033[1;35m"
	colorBoldCyan    = "\033[1;36m"
	colorBoldWhite   = "\033[1;37m"

	// Background colors
	colorBgBlack   = "\033[40m"
	colorBgRed     = "\033[41m"
	colorBgGreen   = "\033[42m"
	colorBgYellow  = "\033[43m"
	colorBgBlue    = "\033[44m"
	colorBgMagenta = "\033[45m"
	colorBgCyan    = "\033[46m"
	colorBgWhite   = "\033[47m"
)

// NewColorProvider creates a new color provider.
func NewColorProvider(enabled bool) *ColorProvider {
	// Auto-detect color support if not explicitly disabled
	if enabled {
		enabled = supportsColor()
	}

	return &ColorProvider{
		enabled: enabled,
		scheme:  DefaultColorScheme(),
	}
}

// DefaultColorScheme returns the default color scheme.
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		Primary:   colorBoldBlue,
		Secondary: colorCyan,
		Success:   colorGreen,
		Warning:   colorYellow,
		Error:     colorRed,
		Info:      colorBlue,
		Muted:     colorBlack,
	}
}

// SetScheme sets the color scheme.
func (c *ColorProvider) SetScheme(scheme ColorScheme) {
	c.scheme = scheme
}

// Enable enables color output.
func (c *ColorProvider) Enable() {
	c.enabled = true
}

// Disable disables color output.
func (c *ColorProvider) Disable() {
	c.enabled = false
}

// IsEnabled returns whether color output is enabled.
func (c *ColorProvider) IsEnabled() bool {
	return c.enabled
}

// colorize applies color to text.
func (c *ColorProvider) colorize(text, color string) string {
	if !c.enabled || color == "" {
		return text
	}
	return color + text + colorReset
}

// Primary applies primary color.
func (c *ColorProvider) Primary(text string) string {
	return c.colorize(text, c.scheme.Primary)
}

// Secondary applies secondary color.
func (c *ColorProvider) Secondary(text string) string {
	return c.colorize(text, c.scheme.Secondary)
}

// Success applies success color.
func (c *ColorProvider) Success(text string) string {
	return c.colorize(text, c.scheme.Success)
}

// Warning applies warning color.
func (c *ColorProvider) Warning(text string) string {
	return c.colorize(text, c.scheme.Warning)
}

// Error applies error color.
func (c *ColorProvider) Error(text string) string {
	return c.colorize(text, c.scheme.Error)
}

// Info applies info color.
func (c *ColorProvider) Info(text string) string {
	return c.colorize(text, c.scheme.Info)
}

// Muted applies muted color.
func (c *ColorProvider) Muted(text string) string {
	return c.colorize(text, c.scheme.Muted)
}

// Bold applies bold formatting.
func (c *ColorProvider) Bold(text string) string {
	if !c.enabled {
		return text
	}
	return "\033[1m" + text + colorReset
}

// Underline applies underline formatting.
func (c *ColorProvider) Underline(text string) string {
	if !c.enabled {
		return text
	}
	return "\033[4m" + text + colorReset
}

// Italic applies italic formatting.
func (c *ColorProvider) Italic(text string) string {
	if !c.enabled {
		return text
	}
	return "\033[3m" + text + colorReset
}

// Strikethrough applies strikethrough formatting.
func (c *ColorProvider) Strikethrough(text string) string {
	if !c.enabled {
		return text
	}
	return "\033[9m" + text + colorReset
}

// RGB applies RGB color.
func (c *ColorProvider) RGB(text string, r, g, b int) string {
	if !c.enabled {
		return text
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s%s", r, g, b, text, colorReset)
}

// BgRGB applies RGB background color.
func (c *ColorProvider) BgRGB(text string, r, g, b int) string {
	if !c.enabled {
		return text
	}
	return fmt.Sprintf("\033[48;2;%d;%d;%dm%s%s", r, g, b, text, colorReset)
}

// Hex applies hex color.
func (c *ColorProvider) Hex(text, hex string) string {
	if !c.enabled {
		return text
	}

	r, g, b, err := parseHex(hex)
	if err != nil {
		return text
	}

	return c.RGB(text, r, g, b)
}

// BgHex applies hex background color.
func (c *ColorProvider) BgHex(text, hex string) string {
	if !c.enabled {
		return text
	}

	r, g, b, err := parseHex(hex)
	if err != nil {
		return text
	}

	return c.BgRGB(text, r, g, b)
}

// Strip removes color codes from text.
func (c *ColorProvider) Strip(text string) string {
	// Simple regex would be better, but avoiding dependencies
	result := text

	// Remove common ANSI escape sequences
	sequences := []string{
		colorReset,
		colorBlack, colorRed, colorGreen, colorYellow,
		colorBlue, colorMagenta, colorCyan, colorWhite,
		colorBoldBlack, colorBoldRed, colorBoldGreen, colorBoldYellow,
		colorBoldBlue, colorBoldMagenta, colorBoldCyan, colorBoldWhite,
		colorBgBlack, colorBgRed, colorBgGreen, colorBgYellow,
		colorBgBlue, colorBgMagenta, colorBgCyan, colorBgWhite,
		"\033[1m", "\033[3m", "\033[4m", "\033[9m",
	}

	for _, seq := range sequences {
		result = strings.ReplaceAll(result, seq, "")
	}

	return result
}

// ColorizeJSON colorizes JSON output.
func (c *ColorProvider) ColorizeJSON(json string) string {
	if !c.enabled {
		return json
	}

	// Simple JSON colorization
	result := json

	// Colorize strings (values in quotes)
	result = strings.ReplaceAll(result, `"`, c.Secondary(`"`))

	// Colorize numbers
	for _, char := range "0123456789" {
		result = strings.ReplaceAll(result, string(char), c.Info(string(char)))
	}

	// Colorize booleans
	result = strings.ReplaceAll(result, "true", c.Success("true"))
	result = strings.ReplaceAll(result, "false", c.Error("false"))

	// Colorize null
	result = strings.ReplaceAll(result, "null", c.Muted("null"))

	// Colorize punctuation
	for _, char := range "{}[],:." {
		result = strings.ReplaceAll(result, string(char), c.Primary(string(char)))
	}

	return result
}

// ColorizeYAML colorizes YAML output.
func (c *ColorProvider) ColorizeYAML(yaml string) string {
	if !c.enabled {
		return yaml
	}

	lines := strings.Split(yaml, "\n")
	var result []string

	for _, line := range lines {
		colorized := line

		// Colorize keys (text before colon)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]

				// Colorize key
				colorized = c.Primary(key) + ":" + value
			}
		}

		// Colorize comments
		if strings.Contains(colorized, "#") {
			parts := strings.SplitN(colorized, "#", 2)
			if len(parts) == 2 {
				colorized = parts[0] + c.Muted("#"+parts[1])
			}
		}

		result = append(result, colorized)
	}

	return strings.Join(result, "\n")
}

// supportsColor detects if the terminal supports color output.
func supportsColor() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check FORCE_COLOR environment variable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Check if stdout is a terminal
	if !isTerminal() {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}

	// Check for color support in TERM
	colorTerms := []string{
		"xterm", "xterm-color", "xterm-256color",
		"screen", "screen-256color",
		"tmux", "tmux-256color",
		"rxvt", "rxvt-unicode", "rxvt-unicode-256color",
		"linux", "cygwin",
	}

	for _, colorTerm := range colorTerms {
		if strings.Contains(term, colorTerm) {
			return true
		}
	}

	// Check COLORTERM environment variable
	if os.Getenv("COLORTERM") != "" {
		return true
	}

	return false
}

// isTerminal checks if the output is a terminal.
func isTerminal() bool {
	// Simple check for terminal
	// In a real implementation, you'd use more sophisticated detection
	return os.Getenv("TERM") != ""
}

// parseHex parses a hex color string.
func parseHex(hex string) (int, int, int, error) {
	// Remove # prefix if present
	if strings.HasPrefix(hex, "#") {
		hex = hex[1:]
	}

	// Ensure hex is 6 characters
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %s", hex)
	}

	// Parse RGB components
	r, err := strconv.ParseInt(hex[0:2], 16, 0)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid red component: %w", err)
	}

	g, err := strconv.ParseInt(hex[2:4], 16, 0)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid green component: %w", err)
	}

	b, err := strconv.ParseInt(hex[4:6], 16, 0)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid blue component: %w", err)
	}

	return int(r), int(g), int(b), nil
}

// Default color provider instance
var defaultColorProvider *ColorProvider

// SetDefaultColorProvider sets the default color provider.
func SetDefaultColorProvider(provider *ColorProvider) {
	defaultColorProvider = provider
}

// GetDefaultColorProvider returns the default color provider.
func GetDefaultColorProvider() *ColorProvider {
	if defaultColorProvider == nil {
		defaultColorProvider = NewColorProvider(true)
	}
	return defaultColorProvider
}

// Color functions using default provider

// Primary applies primary color using default provider.
func Primary(text string) string {
	return GetDefaultColorProvider().Primary(text)
}

// Secondary applies secondary color using default provider.
func Secondary(text string) string {
	return GetDefaultColorProvider().Secondary(text)
}

// Success applies success color using default provider.
func Success(text string) string {
	return GetDefaultColorProvider().Success(text)
}

// Warning applies warning color using default provider.
func Warning(text string) string {
	return GetDefaultColorProvider().Warning(text)
}

// Error applies error color using default provider.
func Error(text string) string {
	return GetDefaultColorProvider().Error(text)
}

// Info applies info color using default provider.
func Info(text string) string {
	return GetDefaultColorProvider().Info(text)
}

// Muted applies muted color using default provider.
func Muted(text string) string {
	return GetDefaultColorProvider().Muted(text)
}

// Bold applies bold formatting using default provider.
func Bold(text string) string {
	return GetDefaultColorProvider().Bold(text)
}
