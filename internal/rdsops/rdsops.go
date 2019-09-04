package rdsops

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/bluebenno/RDS-snapshot-copier/internal/wiring"
	"go.uber.org/zap"
)

// AntiRateLimit will be slept in key places, to preclude AWS API rate-limiting.
const AntiRateLimit = (10 * time.Millisecond)

// List returns all RDS instances in the region. Max of 50000
func List(rdssession rdsiface.RDSAPI) ([]*rds.DBInstance, error) {
	var results []*rds.DBInstance

	params := &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int64(50),
	}

	pageNum := 0
	err := rdssession.DescribeDBInstancesPages(params,
		func(r *rds.DescribeDBInstancesOutput, lastPage bool) bool {
			pageNum++
			results = append(results, r.DBInstances...)
			time.Sleep(AntiRateLimit)
			return pageNum <= 1000
		})

	return results, err
}

// GetTag returns an AWS RDS Tag value, given the Key. Otherwise returns empty string
func GetTag(rdssession rdsiface.RDSAPI, arn, searchKey string) (string, error) {
	c := &rds.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	}
	res, err := rdssession.ListTagsForResource(c)
	if err != nil {
		return "", err
	}

	for _, i := range res.TagList {
		if *i.Key == searchKey {
			return *i.Value, nil
		}
	}
	return "", nil
}

// Filter takes a list of RDS and indentifies the ones that need their snapshots copied
// It does this by checking for the user supplied tag
func Filter(logger *zap.Logger, cfg *wiring.Config, rdssession rdsiface.RDSAPI, input []*rds.DBInstance) ([]*rds.DBInstance, error) {
	var filtered []*rds.DBInstance
	for _, i := range input {
		if *i.DBInstanceStatus != "available" {
			logger.Info("Skipping RDS with status != available", zap.String("instance", *i.DBInstanceIdentifier))
			continue
		}

		t, err := GetTag(rdssession, *i.DBInstanceArn, cfg.Tag)
		if err != nil {
			logger.Warn("Error encountered when checking AWS tags", zap.Any("instance", *i.DBInstanceIdentifier), zap.Error(err))
			continue
		}

		if t == "" {
			continue
		}

		logger.Info("found in scope RDS", zap.String("instance", *i.DBInstanceIdentifier))
		filtered = append(filtered, i)

		//
		time.Sleep(AntiRateLimit)
	}

	return filtered, nil
}
