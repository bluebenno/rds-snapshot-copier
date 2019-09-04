package wiring

// Config defines the app config
type Config struct {
	DryRun          bool
	Tag             string // An AWS Tag on the RDS, which will flag copying of the snapshots
	LogLevel        string
	MaxCopyInFlight int
	MaxSnap         int
	RunEvery        int
	SourceRegion    string
	TargetKMS       string
	TargetRegion    string
}
