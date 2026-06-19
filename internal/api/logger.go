package api

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger for structured logging with our required format
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new Logger instance with JSON handler
func NewLogger() *Logger {
	// Read log level from environment, default to INFO
	logLevelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if logLevelStr == "" {
		logLevelStr = "INFO"
	}

	var logLevel slog.Level
	switch logLevelStr {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)

	return &Logger{logger: logger}
}

// LogCreate logs a successful purchase creation
func (l *Logger) LogCreate(ctx context.Context, purchaseID string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "api_handler"),
		slog.String("purchase_id", purchaseID),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, "purchase created", attrs...)
}

// LogCreateError logs a purchase creation error
func (l *Logger) LogCreateError(ctx context.Context, errorCode, message string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "api_handler"),
		slog.String("error_code", errorCode),
		slog.String("message", message),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelError, "purchase creation failed", attrs...)
}

// LogRetrieve logs a successful purchase retrieval
func (l *Logger) LogRetrieve(ctx context.Context, purchaseID string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "api_handler"),
		slog.String("purchase_id", purchaseID),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, "purchase retrieved", attrs...)
}

// LogRetrieveError logs a purchase retrieval error
func (l *Logger) LogRetrieveError(ctx context.Context, purchaseID, errorCode, message string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "api_handler"),
		slog.String("purchase_id", purchaseID),
		slog.String("error_code", errorCode),
		slog.String("message", message),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelError, "purchase retrieval failed", attrs...)
}

// LogConversion logs a successful currency conversion
func (l *Logger) LogConversion(ctx context.Context, purchaseID, currency, convertedAmount string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "api_handler"),
		slog.String("purchase_id", purchaseID),
		slog.String("currency", currency),
		slog.String("converted_amount", convertedAmount),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, "purchase converted", attrs...)
}

// LogConversionError logs a currency conversion error
func (l *Logger) LogConversionError(ctx context.Context, purchaseID, currency, errorCode, message string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "api_handler"),
		slog.String("purchase_id", purchaseID),
		slog.String("currency", currency),
		slog.String("error_code", errorCode),
		slog.String("message", message),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelError, "currency conversion failed", attrs...)
}

// LogTreasuryAPIQuery logs a Treasury API rate lookup
func (l *Logger) LogTreasuryAPIQuery(ctx context.Context, currency, purchaseDate, purchaseID string) {
	attrs := []slog.Attr{
		slog.String("component", "purchase_service"),
		slog.String("currency", currency),
		slog.String("purchase_date", purchaseDate),
		slog.String("purchase_id", purchaseID),
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, "querying treasury api for exchange rate", attrs...)
}

// LogInfo logs an informational message with structured context
func (l *Logger) LogInfo(ctx context.Context, component, message string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", component),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogError logs an error message with structured context
func (l *Logger) LogError(ctx context.Context, component, message, errorCode string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", component),
		slog.String("error_code", errorCode),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// Info logs an informational message for health checks
func (l *Logger) Info(message string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "health"),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(context.Background(), slog.LevelInfo, message, attrs...)
}

// Error logs an error message for health checks
func (l *Logger) Error(message string, metadata map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("component", "health"),
	}

	for k, v := range metadata {
		attrs = append(attrs, slog.Any(k, v))
	}

	l.logger.LogAttrs(context.Background(), slog.LevelError, message, attrs...)
}
