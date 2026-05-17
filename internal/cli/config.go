package cli

type Config struct {
	FilePath      string
	Query         string
	TUI           bool
	ShowHelp      bool
	ShowVersion   bool
	JsonFormatter bool
	Pretty        bool
}
