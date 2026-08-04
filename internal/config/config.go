package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mymeetily/mymeetily/internal/layout"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ASR    ASRConfig    `mapstructure:"asr" yaml:"asr"`
	Ollama OllamaConfig `mapstructure:"ollama" yaml:"ollama"`
	Audio  AudioConfig  `mapstructure:"audio" yaml:"audio"`
	Output OutputConfig `mapstructure:"output" yaml:"output"`
}

type ASRConfig struct {
	WhisperBinary string `mapstructure:"whisper_binary" yaml:"whisper_binary"`
	ModelPath     string `mapstructure:"model_path" yaml:"model_path"`
	Language      string `mapstructure:"language" yaml:"language"`
}

type OllamaConfig struct {
	Endpoint    string  `mapstructure:"endpoint" yaml:"endpoint"`
	Model       string  `mapstructure:"model" yaml:"model"`
	Temperature float64 `mapstructure:"temperature" yaml:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens" yaml:"max_tokens"`
}

type AudioConfig struct {
	Backend    string `mapstructure:"backend" yaml:"backend"`
	SampleRate int    `mapstructure:"sample_rate" yaml:"sample_rate"`
	Channels   int    `mapstructure:"channels" yaml:"channels"`
}

type OutputConfig struct {
	OutputDir string `mapstructure:"output_dir" yaml:"output_dir"`
}

func Load(cfgPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("asr.whisper_binary", layout.WhisperBinary)
	v.SetDefault("asr.model_path", layout.WhisperModel)
	v.SetDefault("asr.language", "zh")
	v.SetDefault("ollama.endpoint", "http://127.0.0.1:11434")
	v.SetDefault("ollama.model", "qwen2.5:1.5b-instruct")
	v.SetDefault("ollama.temperature", 0.3)
	v.SetDefault("ollama.max_tokens", 4096)
	v.SetDefault("audio.backend", "wasapi")
	v.SetDefault("audio.sample_rate", 16000)
	v.SetDefault("audio.channels", 1)
	v.SetDefault("output.output_dir", "./output")

	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// UserConfigPath returns the per-user settings path. Keeping preferences out of
// the installation directory means application updates do not overwrite them.
func UserConfigPath() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(baseDir, "MyMeetily", "config.yaml"), nil
}

// LoadWithUserOverrides loads the bundled defaults and overlays the user's
// saved preferences when present.
func LoadWithUserOverrides() (*Config, error) {
	cfg, err := Load("")
	if err != nil {
		return nil, err
	}

	userPath, err := UserConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(userPath)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read user config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse user config: %w", err)
	}
	return cfg, nil
}

// SaveUser persists the current settings for the signed-in Windows user
// without modifying the repository's bundled config.yaml.
func SaveUser(cfg *Config) error {
	userPath, err := UserConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		return fmt.Errorf("create user config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode user config: %w", err)
	}
	// Write directly because os.Rename cannot replace an existing destination
	// consistently on Windows. This also makes repeated saves reliable.
	if err := os.WriteFile(userPath, data, 0o600); err != nil {
		return fmt.Errorf("write user config: %w", err)
	}
	return nil
}
