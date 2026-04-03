package config

type Config struct {
	Log     LogConfig     `yaml:"log"`
	Audit   AuditConfig   `yaml:"audit"`
	Scanner ScannerConfig `yaml:"scanner"`
	Storage StorageConfig `yaml:"storage"`
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
	DataDir           string `yaml:"data_dir"`
	ProcessSystemFile string `yaml:"process_system_file"`
	FileSystemFile    string `yaml:"file_system_file"`
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
