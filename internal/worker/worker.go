package worker

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"

	"go.uber.org/zap"

	"github.com/bluebenno/rds-snapshot-copier/internal/rdsops"
	"github.com/bluebenno/rds-snapshot-copier/internal/snapops"
	"github.com/bluebenno/rds-snapshot-copier/internal/wiring"
)

// Run wires things together and will start the infinite loop
func Run(logger *zap.Logger, cfg *wiring.Config) error {
	return Looper(logger, cfg)
}

// Looper is an infinite loop.  Each loop will:
// 1) Identify and then copy the snapshots from the source to the target region
// 2) Optionally, encrypt the snapshots at the target region, with a supplied KMS key
// 3) Optionally, housekeep snapshots at the target region
func Looper(logger *zap.Logger, cfg *wiring.Config) error {
	for {
		SrcRDSSource, err := wiring.Session(cfg, cfg.SourceRegion)
		if err != nil {
			logger.Fatal("Failed to create an AWS rds Session for the source region", zap.String("source_region", cfg.SourceRegion), zap.Error(err))
		}
		SrcRDSTarget, err := wiring.Session(cfg, cfg.TargetRegion)
		if err != nil {
			logger.Fatal("Failed to create an AWS rds Session for the target region", zap.String("target_region", cfg.TargetRegion), zap.Error(err))
		}

		AllSourceRDS, err := rdsops.List(SrcRDSSource)
		if err != nil {
			logger.Fatal("Failed to get a list of rds Instances", zap.String("source_region", cfg.SourceRegion), zap.Error(err))
		}

		inscopeRDS, err := rdsops.Filter(logger, cfg, SrcRDSSource, AllSourceRDS)
		if err != nil {
			logger.Fatal("Failed to find inscope rds Instances", zap.String("source_region", cfg.SourceRegion), zap.Error(err))
		}

		ssq, err := buildQueue(logger, cfg, SrcRDSSource, SrcRDSTarget, inscopeRDS)
		if err != nil {
			logger.Fatal("Failed to build a list of snapshots to copy", zap.String("source_region", cfg.SourceRegion), zap.Error(err))
		}

		// The following will block until completed
		num, _ := copySnapShots(logger, cfg, SrcRDSSource, SrcRDSTarget, ssq)
		println(num)
		os.Exit(1)

		// Sleep to next run
		time.Sleep(time.Duration(cfg.RunEvery) * time.Minute)
	}
}

func copySnapShots(logger *zap.Logger, cfg *wiring.Config, srcRDSSource rdsiface.RDSAPI, srcRDSTarget rdsiface.RDSAPI, snaps []*rds.DBSnapshot) (int, error) {

	type copyjob struct {
		cfg          *wiring.Config
		logger       *zap.Logger
		snapshot     *rds.DBSnapshot
		srcRDSSource rdsiface.RDSAPI
		srcRDSTarget rdsiface.RDSAPI
	}

	type result struct {
		worker int
		start  time.Time
		finish time.Time
		result int
	}

	var results []result
	var mu sync.Mutex
	var wg sync.WaitGroup
	ch := make(chan copyjob)

	for i := 1; i <= cfg.MaxCopyInFlight; i++ {
		go func(i int) {
			for j := range ch {
				myresult := result{worker: i, start: time.Now()}
				tName := strings.Replace((*j.snapshot.DBSnapshotIdentifier + "-copyfrom-" + j.cfg.SourceRegion), "rds:", "", -1)

				res, err := copySnap(cfg, srcRDSSource, srcRDSTarget, j.snapshot)
				if err != nil {
					logger.Warn("Failed to perform snapshot pull", zap.String("source_region", cfg.SourceRegion), zap.String("target_region", cfg.TargetRegion),
						zap.String("rds", *j.snapshot.DBInstanceIdentifier), zap.String("snapshot", *j.snapshot.DBSnapshotIdentifier), zap.Error(err))
					continue
				} else {
					logger.Info("Snapshot copy started", zap.String("source_region", cfg.SourceRegion), zap.String("target_region", cfg.TargetRegion),
						zap.String("rds", *j.snapshot.DBInstanceIdentifier), zap.String("source_snapshot", *j.snapshot.DBSnapshotIdentifier), zap.String("target_snapshot", tName), zap.Any("result", *res.DBSnapshot))
				}

				// poll until AWS has copied the snapshot has finished, this could be a long time if it is a very big/busy database
				for {
					status, err := snapops.Describe(srcRDSTarget, *res.DBSnapshot.DBSnapshotIdentifier)
					if err != nil {
						logger.Warn("Failed get status on nearly created sanpshot", zap.String("source_region", cfg.SourceRegion), zap.String("target_region", cfg.TargetRegion),
							zap.String("rds", *j.snapshot.DBInstanceIdentifier), zap.String("snapshot", *j.snapshot.DBSnapshotIdentifier), zap.Error(err))
					}
					time.Sleep(10 * time.Second)
					if *status.Status == "available" {
						break
					}

				}
				myresult.finish = time.Now()
				myresult.result = 0

				mu.Lock()
				results = append(results, myresult)
				mu.Unlock()
				wg.Done()
			}
		}(i)
	}

	for _, s := range snaps {
		wg.Add(1)
		ch <- copyjob{
			cfg:          cfg,
			logger:       logger,
			snapshot:     s,
			srcRDSSource: srcRDSSource,
			srcRDSTarget: srcRDSTarget,
		}
	}
	close(ch)
	wg.Wait()

	return 0, nil
}

// Copy a single snapshot
func copySnap(cfg *wiring.Config, srcRDSSource rdsiface.RDSAPI, srcRDSTarget rdsiface.RDSAPI, s *rds.DBSnapshot) (*rds.CopyDBSnapshotOutput, error) {

	tName := strings.Replace((*s.DBSnapshotIdentifier + "-cf-" + cfg.SourceRegion), "rds:", "", -1)

	var res *rds.CopyDBSnapshotOutput
	var err error

	if cfg.TargetKMS != "" {
		res, err = snapops.PullEncryptedSnapShot(cfg, srcRDSSource, srcRDSTarget, s.DBSnapshotArn, tName)
	} else {
		res, err = snapops.PullSnapShot(cfg, srcRDSTarget, s.DBSnapshotArn, tName)
	}

	if err != nil {
		return nil, err
	}

	return res, nil
}

// buildQueue will build a list of the (latest) snapshots for each rds
func buildQueue(logger *zap.Logger, cfg *wiring.Config, rdssession rdsiface.RDSAPI, srcRDSTarget rdsiface.RDSAPI, isr []*rds.DBInstance) ([]*rds.DBSnapshot, error) {
	var toCopy []*rds.DBSnapshot

	for _, i := range isr {
		logger.Info("Looking at rds", zap.String("RDS", *i.DBInstanceIdentifier))

		lsSource, err := snapops.List(rdssession, *i.DBInstanceIdentifier)
		if err != nil {
			logger.Warn("Failed to list snapshots", zap.String("region", cfg.SourceRegion), zap.String("rds", *i.DBInstanceIdentifier), zap.Error(err))
			continue
		}

		latestS, err := snapops.GetLatest(lsSource)
		if err != nil {
			logger.Warn("Failed to find latest snapshot", zap.String("region", cfg.SourceRegion), zap.String("rds", *i.DBInstanceIdentifier), zap.Error(err))
			continue
		}

		if latestS == nil {
			logger.Info("No source snapshots found", zap.String("region", cfg.SourceRegion), zap.String("rds", *i.DBInstanceIdentifier))
			continue
		}

		// Has it already been copied?
		tName := strings.Replace((*latestS.DBSnapshotIdentifier + "-cf-" + cfg.SourceRegion), "rds:", "", -1)
		exists, err := snapops.Describe(srcRDSTarget, tName)
		if err != nil {
			logger.Warn("Failed to search for snapshot at target region", zap.String("region", cfg.TargetRegion), zap.String("snapshot", tName), zap.Error(err))
			continue
		}

		if exists != nil {
			logger.Info("Snapshot already found in target region", zap.String("region", cfg.TargetRegion), zap.String("snapshot", tName), zap.Error(err))
			continue
		}

		logger.Info("enqueue snapshot for copy", zap.String("region", cfg.SourceRegion), zap.String("rds", *i.DBInstanceIdentifier), zap.String("snapshot", *latestS.DBSnapshotIdentifier))
		toCopy = append(toCopy, latestS)
	}
	return toCopy, nil
}
