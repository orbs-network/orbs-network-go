package log

type Output interface {
	Append(level string, message string, fields ...*Field)
}
