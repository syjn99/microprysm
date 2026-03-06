// Package common provides shared CLI setup logic for Prysm binaries.
package common

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/OffchainLabs/prysm/v7/cmd"
	"github.com/OffchainLabs/prysm/v7/config/features"
	"github.com/OffchainLabs/prysm/v7/io/logs"
	"github.com/OffchainLabs/prysm/v7/monitoring/journald"
	"github.com/OffchainLabs/prysm/v7/runtime/debug"
	prefixed "github.com/OffchainLabs/prysm/v7/runtime/logging/logrus-prefixed-formatter"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Before returns a cli.BeforeFunc that configures logging, debug, features, etc.
// Parameters:
//   - appName: name used in the startup log message (e.g. "Prysm Beacon Chain", "Prysm Validator")
//   - appFlags: the full set of flags for the app (used for config file loading)
//   - ephemeralLogDisableFlag: the flag that controls ephemeral log file (differs between BN/VC packages)
func Before(appName string, appFlags []cli.Flag, ephemeralLogDisableFlag *cli.BoolFlag) cli.BeforeFunc {
	return func(ctx *cli.Context) error {
		// Load flags from config file, if specified.
		if err := cmd.LoadFlagsFromConfig(ctx, appFlags); err != nil {
			return errors.Wrap(err, "failed to load flags from config file")
		}

		// determine default log verbosity
		verbosity := ctx.String(cmd.VerbosityFlag.Name)
		verbosityLevel, err := logrus.ParseLevel(verbosity)
		if err != nil {
			return errors.Wrap(err, "failed to parse log verbosity")
		}

		// determine per package verbosity. if not set, maxLevel will be 0.
		vmoduleInput := strings.Join(ctx.StringSlice(cmd.LogVModuleFlag.Name), ",")
		vmodule, maxLevel, err := cmd.ParseVModule(vmoduleInput)
		if err != nil {
			return errors.Wrap(err, "failed to parse log vmodule")
		}

		// set the global logging level and data
		logs.SetLoggingLevelAndData(verbosityLevel, vmodule, maxLevel, ctx.Bool(ephemeralLogDisableFlag.Name))

		format := ctx.String(cmd.LogFormat.Name)
		switch format {
		case "text":
			// disabling logrus default output so we can control it via different hooks
			logrus.SetOutput(io.Discard)

			// create a custom formatter and hook for terminal output
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05.00"
			formatter.FullTimestamp = true
			formatter.ForceFormatting = true
			formatter.ForceColors = true
			formatter.VModule = vmodule
			formatter.BaseVerbosity = verbosityLevel

			logrus.AddHook(&logs.WriterHook{
				Formatter:     formatter,
				Writer:        os.Stderr,
				AllowedLevels: logrus.AllLevels[:max(verbosityLevel, maxLevel)+1],
				Identifier:    logs.LogTargetUser,
			})
		case "fluentd":
			f := joonix.NewFormatter()
			if err := joonix.DisableTimestampFormat(f); err != nil {
				panic(err) // lint:nopanic -- This shouldn't happen, but crashing immediately at startup is OK.
			}
			logrus.SetFormatter(f)
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05.00",
			})
		case "journald":
			if err := journald.Enable(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown log format %s", format)
		}

		logFileName := ctx.String(cmd.LogFileName.Name)
		if logFileName != "" {
			if err := logs.ConfigurePersistentLogging(logFileName, format, verbosityLevel, vmodule); err != nil {
				log.WithError(err).Error("Failed to configuring logging to disk.")
			}
		}

		if !ctx.Bool(ephemeralLogDisableFlag.Name) {
			if err := logs.ConfigureEphemeralLogFile(ctx.String(cmd.DataDirFlag.Name), ctx.App.Name); err != nil {
				log.WithError(err).Error("Failed to configure debug log file")
			}
		}

		// Log Prysm version on startup. After initializing log-file and ephemeral log-file.
		log.WithFields(logrus.Fields{
			"version": version.Version(),
		}).Infof("%s started", appName)

		if err := debug.Setup(ctx); err != nil {
			return errors.Wrap(err, "failed to setup debug")
		}

		if err := features.ValidateNetworkFlags(ctx); err != nil {
			return errors.Wrap(err, "provided multiple network flags")
		}

		return cmd.ValidateNoArgs(ctx)
	}
}
