package config

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
	"github.com/stoewer/go-strcase"
)

const (
	defaultMetricsBindAddress     = ":8080"
	defaultHealthProbeBindAddress = ":8081"
	defaultNodeSelector           = ""
	defaultMinLifetime            = 45 * time.Minute
	defaultLeaderElectID          = "node-min-lifetime.eplight.org"
)

type Config struct {
	MetricsBindAddress     string
	HealthProbeBindAddress string

	NodeSelector       KubernetesSelector
	DefaultMinLifetime time.Duration
	LeaderElect        bool
	LeaderElectID      string
}

func MustLoadConfig() *Config {
	defaults := map[string]any{
		"metrics-bind-address":      defaultMetricsBindAddress,
		"health-probe-bind-address": defaultHealthProbeBindAddress,
		"node-selector":             defaultNodeSelector,
		"default-min-lifetime":      defaultMinLifetime,
		"leader-elect":              false,
		"leader-elect-id":           defaultLeaderElectID,
	}

	fs := pflag.NewFlagSet("config", pflag.ExitOnError)
	fs.String("metrics-bind-address", defaultMetricsBindAddress, "The address the metric endpoint binds to.")
	fs.String("health-probe-bind-address", defaultHealthProbeBindAddress, "The address the probe endpoint binds to.")
	fs.String("node-selector", defaultNodeSelector, "Optional selector to limit which nodes are considered.")
	fs.Duration("default-min-lifetime", defaultMinLifetime, "Default minimum lifetime of nodes.")
	fs.Bool("leader-elect", false, "Enable leader election for controllers manager.")
	fs.String("leader-elect-id", defaultLeaderElectID, "ID to use for leader election.")
	fs.Parse(os.Args[1:])

	k := koanf.New(".")

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		slog.Error("Could not load default config", "error", err)
		os.Exit(1)
	}
	if err := k.Load(env.Provider(".", env.Opt{
		TransformFunc: func(k, v string) (string, any) {
			return strings.ReplaceAll("_", "-", strings.ToLower(k)), v
		},
	}), nil); err != nil {
		slog.Error("Could not load env config", "error", err)
		os.Exit(1)
	}
	if err := k.Load(posflag.Provider(fs, ".", k), nil); err != nil {
		slog.Error("Could not load flags", "error", err)
		os.Exit(1)
	}

	out := &Config{}
	if err := k.UnmarshalWithConf("", &out, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			MapFieldName: func(s string) string {
				return strcase.KebabCase(s)
			},
		},
	}); err != nil {
		slog.Error("Could not unmarshal config", "error", err)
		os.Exit(1)
	}

	return out
}
