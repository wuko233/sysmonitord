package config

type Config struct {
	Log          LogConfig          `yaml:"log"`
	Audit        AuditConfig        `yaml:"audit"`
	Scanner      ScannerConfig      `yaml:"scanner"`
	Storage      StorageConfig      `yaml:"storage"`
	Notification NotificationConfig `yaml:"notification"`
}

type NotificationConfig struct {
	Email    EmailConfig `yaml:"email"`
	Interval int         `yaml:"interval"`
}

type EmailConfig struct {
	Enabled    bool       `yaml:"enabled"`
	Recipients []string   `yaml:"recipients"`
	SMTP       SMTPConfig `yaml:"smtp"`
}

type SMTPConfig struct {
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type AuditConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Server     string `yaml:"server"`
	Port       int    `yaml:"port"`
	BufferSize int    `yaml:"buffer_size"`
}

type ScannerConfig struct {
	File    FileScannerConfig    `yaml:"file"`
	Hash    hashConfig           `yaml:"hash"`
	Process ProcessScannerConfig `yaml:"process"`
}

type hashConfig struct {
	Algorithm string `yaml:"algorithm"`
}

type ProcessScannerConfig struct {
	Interval int `yaml:"interval"`
}

type StorageConfig struct {
	DataDir                string `yaml:"data_dir"`
	ProcessSystemFile      string `yaml:"process_system_file"`
	FileSystemFile         string `yaml:"file_system_file"`
	DubiousFileListFile    string `yaml:"dubious_file_list_file"`
	DubiousProcessListFile string `yaml:"dubious_process_list_file"`
}

type FileScannerConfig struct {
	IncludePaths     []string `yaml:"include_paths"`
	ExcludePaths     []string `yaml:"exclude_paths"`
	FastHash         bool     `yaml:"fast_hash"`
	FastHashSizeRaw  string   `yaml:"fast_hash_size"`
	FastHashChunkRaw string   `yaml:"fast_hash_chunk"`
	FastHashSize     int64
	FastHashChunk    int64
}
