package nana

import "github.com/Sirupsen/logrus"

// LogFormatter Ensures all entries are logged with standard fields for the application
type LogFormatter struct {
	name        string
	version     string
	environment string
	formatter   logrus.Formatter
}

// NewLogFormatter returns a LogFormatter for the specified config.
func NewLogFormatter(config AppConfigurer, formatter logrus.Formatter) *LogFormatter {
	return &LogFormatter{
		name:        config.GetName(),
		version:     config.GetVersion(),
		environment: config.GetEnvironment(),
		formatter:   formatter,
	}
}

// Format adds standard fields to all log output.
func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	wrappedEntry := entry.WithFields(logrus.Fields{
		"application": f.name,
		"version":     f.version,
		"environment": f.environment,
	})
	wrappedEntry.Time = entry.Time
	wrappedEntry.Message = entry.Message
	wrappedEntry.Level = entry.Level
	return f.formatter.Format(wrappedEntry)
}
