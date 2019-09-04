package Flags

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/bluebenno/RDS-snapshot-copier/internal/wiring"
)

// Flags parses the command line flags and or environmental variables
func Flags(name, gitCommit, version string, cfg *wiring.Config) *kingpin.Application {

	app := kingpin.New(name, "An AWS RDS snapshot copier that has region and encryption support")

	app.Flag("dryrun", "do a dry run, print what can be done").Short('d').Envar("DRY_RUN").BoolVar(&cfg.DryRun)
	app.Flag("loglevel", `log level: "debug", "info", "warn", "error", "dpanic", "panic", and "fatal".`).Short('l').Envar("LOG_LEVEL").Default("info").EnumVar(&cfg.LogLevel, "debug", "info", "warn", "error", "dpanic", "panic", "fatal")
	app.Flag("maxinflight", "Maximum copy operations in flight. AWS max is six").Short('f').Default("2").Envar("MAX_SNAPSHOT_FLIGHT").IntVar(&cfg.MaxCopyInFlight)
	app.Flag("maxsnapshots", "Maximum number of Snapshots per RDS, to keep in target region").Short('m').Default("0").Envar("MAX_SNAPSHOT_TARGET").IntVar(&cfg.MaxSnap)
	app.Flag("runevery", "How often should the Source Region be polled for new snapshots, in minutes").Short('r').Default("0").Envar("RUN_EVERY_MINS").IntVar(&cfg.RunEvery)
	app.Flag("sourceregion", "AWS Source Region").Short('s').Envar("SOURCE_REGION").StringVar(&cfg.SourceRegion)
	app.Flag("tag", "RDS with the value tag will have their snapshots copied").Short('a').Envar("TAG").StringVar(&cfg.Tag)
	app.Flag("targetkms", "Encrypt the snapshot at the target with KMS key").Short('k').Default("").Envar("TARGET_KMS").StringVar(&cfg.TargetKMS)
	app.Flag("targetregion", "AWS Target Region").Short('t').Envar("TARGET_REGION").StringVar(&cfg.TargetRegion)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	return app
}
