package cli

type Config struct {
	FilePath      string
	Query         string
	Columns       []string
	TUI           bool
	Table         bool
	ShowHelp      bool
	ShowVersion   bool
	JsonFormatter bool
	Pretty        bool
}
