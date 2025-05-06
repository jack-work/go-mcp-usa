package logging

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggingOptFunc func(*LoggingOpts)

func defaultOpts() LoggingOpts {
	return LoggingOpts{
		Filename:   GetLogFilePath("application"),
		MaxSize:    100,
		MaxAge:     14,
		MaxBackups: 3,
		Compress:   true,
	}
}

type LoggingOpts struct {
	ServiceName string
	Filename    string
	MaxSize     int
	MaxAge      int
	MaxBackups  int
	Compress    bool
}

func WithServiceName(name string) LoggingOptFunc {
	return func(opts *LoggingOpts) {
		opts.Filename = GetLogFilePath(name)
		opts.ServiceName = name
	}
}

func InitTracer(opts ...LoggingOptFunc) (*trace.TracerProvider, error) {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}
	fileWriter := &lumberjack.Logger{
		Filename:   o.Filename,
		MaxSize:    o.MaxSize,
		MaxAge:     o.MaxAge,
		MaxBackups: o.MaxBackups,
		Compress:   o.Compress,
	}

	// both for now but once it's working then file only
	// writer := io.MultiWriter(os.Stdout, fileWriter)
	exporter, err := stdouttrace.New(
		// stdouttrace.WithWriter(writer),
		stdouttrace.WithWriter(fileWriter),
		stdouttrace.WithPrettyPrint())

	if err != nil {
		return nil, err
	}

	// Create a resource describing the service
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(o.ServiceName),
		semconv.ServiceVersionKey.String("0.0.1"),
	)

	// Create the trace provider with the exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set the global trace provider
	otel.SetTracerProvider(tp)

	return tp, nil
}

func GetLogFilePath(appName string) string {
	// Check for environment variable override first
	if envLogPath := os.Getenv("FIGARO_LOG_PATH"); envLogPath != "" {
		return envLogPath
	}

	var basePath string

	switch runtime.GOOS {
	case "windows":
		// Windows path handling (unchanged)
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		basePath = filepath.Join(appData, appName, "logs")
	case "darwin":
		// macOS path handling (unchanged)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}
		basePath = filepath.Join(homeDir, "Library", "Logs", appName)
	default:
		// Linux/NixOS handling
		// First check XDG_STATE_HOME (NixOS often sets this)
		xdgStateHome := os.Getenv("XDG_STATE_HOME")
		if xdgStateHome != "" {
			basePath = filepath.Join(xdgStateHome, appName, "logs")
		} else {
			// Check if running in NixOS's systemd service context
			if os.Getenv("RUNTIME_DIRECTORY") != "" {
				// If running as a service, use the runtime directory
				basePath = filepath.Join(os.Getenv("RUNTIME_DIRECTORY"), "logs")
			} else if os.Getenv("STATE_DIRECTORY") != "" {
				// For persistent state in a service
				basePath = filepath.Join(os.Getenv("STATE_DIRECTORY"), "logs")
			} else {
				// Fall back to XDG spec default location
				homeDir, err := os.UserHomeDir()
				if err != nil {
					log.Fatalf("Failed to get home directory: %v", err)
				}
				basePath = filepath.Join(homeDir, ".local", "state", appName, "logs")
			}
		}
	}

	// Ensure the directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	return filepath.Join(basePath, "application.log")
}
