package snapops

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"

	"github.com/bluebenno/RDS-snapshot-copier/internal/wiring"
)

// AntiRateLimit will be slept in key places, to preclude AWS API rate-limiting.
const AntiRateLimit = (10 * time.Millisecond)

// List will list all the snapshots for a given RDS
func List(rdssession rdsiface.RDSAPI, rdshost string) ([]*rds.DBSnapshot, error) {

	input := &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: aws.String(rdshost),
		IncludePublic:        aws.Bool(false),
		IncludeShared:        aws.Bool(false),
		MaxRecords:           aws.Int64(20),
	}

	var res []*rds.DBSnapshot
	pageNum := 0
	err := rdssession.DescribeDBSnapshotsPages(input,
		func(r *rds.DescribeDBSnapshotsOutput, lastPage bool) bool {
			pageNum++
			res = append(res, r.DBSnapshots...)
			time.Sleep(AntiRateLimit)
			return pageNum <= 1000
		})

	return res, err
}

// Describe will describe a snapshot
func Describe(rdssession rdsiface.RDSAPI, snap string) (*rds.DBSnapshot, error) {

	input := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(snap),
		MaxRecords:           aws.Int64(20), // AWS constraint; min 20
	}

	res, err := rdssession.DescribeDBSnapshots(input)
	if err != nil {
		err, ok := err.(awserr.Error)
		if ok && err.Code() == rds.ErrCodeDBSnapshotNotFoundFault {
			return nil, nil
		}
		return nil, err
	}

	if len(res.DBSnapshots) == 1 {
		return res.DBSnapshots[0], nil
	}

	return nil, err
}

// PullSnapShot pull a copy of an AWS RDS Snapshot from a remote region. It is not blocking.
func PullSnapShot(cfg *wiring.Config, rdssession rdsiface.RDSAPI, arn *string, targetsnapshotname string) (*rds.CopyDBSnapshotOutput, error) {
	input := &rds.CopyDBSnapshotInput{
		SourceDBSnapshotIdentifier: aws.String(*arn),
		TargetDBSnapshotIdentifier: aws.String(targetsnapshotname),
		DestinationRegion:          aws.String(cfg.TargetRegion),
		KmsKeyId:                   aws.String(cfg.TargetKMS),
	}
	result, err := rdssession.CopyDBSnapshot(input)
	return result, err
}

// PullEncryptedSnapShot pulls an encrypted AWS RDS Snapshot from a remote region. It is not blocking.
func PullEncryptedSnapShot(cfg *wiring.Config, rdssessionsource rdsiface.RDSAPI, rdssessiontarget rdsiface.RDSAPI, arn *string, targetsnapshotname string) (*rds.CopyDBSnapshotOutput, error) {
	// Build the PreSignedUrl containing the CopyDBSnapshot API
	inputps := &rds.CopyDBSnapshotInput{
		SourceDBSnapshotIdentifier: aws.String(*arn),
		TargetDBSnapshotIdentifier: aws.String(targetsnapshotname),
		DestinationRegion:          aws.String(cfg.TargetRegion),
		SourceRegion:               aws.String(cfg.SourceRegion),
		KmsKeyId:                   aws.String(cfg.TargetKMS),
	}
	request, _ := rdssessionsource.CopyDBSnapshotRequest(inputps) // _ is output, never populated in this case as it isn't run
	psurl, err := request.Presign(100 * time.Second)
	if err != nil {
		return nil, err
	}

	input := &rds.CopyDBSnapshotInput{
		SourceDBSnapshotIdentifier: aws.String(*arn),
		TargetDBSnapshotIdentifier: aws.String(targetsnapshotname),
		DestinationRegion:          aws.String(cfg.TargetRegion),
		KmsKeyId:                   aws.String(cfg.TargetKMS),
		PreSignedUrl:               aws.String(psurl),
	}
	result, err := rdssessiontarget.CopyDBSnapshot(input)
	return result, err
}

// ListExpired lists the snapshots for an RDS, that are considered expired.
func ListExpired(cfg *wiring.Config, rdssessiontarget rdsiface.RDSAPI, instance *rds.DBInstance) ([]*rds.DBSnapshot, error) {
	ls, err := List(rdssessiontarget, *instance.DBInstanceIdentifier)
	if err != nil {
		return nil, err
	}

	// if have less (or equal) to cfg.MaxSnap; Just return
	if len(ls) <= cfg.MaxSnap {
		return nil, nil
	}

	expired, err := GetSlice(ls, 0, (len(ls) - cfg.MaxSnap))
	if err != nil {
		return nil, err
	}

	return expired, nil
}

// Delete will delete a list of snapshots.
func Delete(rdssessiontarget rdsiface.RDSAPI, snaps []*rds.DBSnapshot) (int, error) {
	if snaps == nil {
		return 0, nil
	}

	var count int
	for _, i := range snaps {
		del := &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: i.DBSnapshotIdentifier,
		}
		r, e := rdssessiontarget.DeleteDBSnapshot(del)
		if e != nil {
			return 0, e
		}

		if *r.DBSnapshot.Status == "deleted" {
			count++
		}

		time.Sleep(AntiRateLimit)
	}
	return count, nil
}

// GetSlice returns a "slice" of a list of snapshots, ordered by date.
// TODO: Yuck the following sucks and needs cleanup!
func GetSlice(all []*rds.DBSnapshot, start, num int) ([]*rds.DBSnapshot, error) {
	if len(all) == 0 {
		return nil, nil
	}
	lena := len(all)
	var end int
	if start >= 0 {
		end = start + num
	} else { // -1
		start = lena + start - num + 1
		end = lena + start
	}
	if start > lena {
		start = lena
	}
	if start < 0 {
		start = 0
	}
	if end > lena {
		end = lena
	}
	return all[start:end], nil
}

// GetLatest returns the most recent snapshot for a RDS
func GetLatest(all []*rds.DBSnapshot) (*rds.DBSnapshot, error) {
	r, e := GetSlice(all, -1, 1)
	if e != nil {
		return nil, e
	}
	if r == nil {
		return nil, nil
	}
	return r[0], nil
}
