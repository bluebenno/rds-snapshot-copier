package wiring

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

// Session initialises a connection for an AWS rds, to a particular region
func Session(cfg *Config, region string) (*rds.RDS, error) {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	rs := rds.New(s)
	if rs != nil {
		return rs, nil
	}
	return nil, fmt.Errorf("failed to initate a Session to the AWS rds endpoint")
}
