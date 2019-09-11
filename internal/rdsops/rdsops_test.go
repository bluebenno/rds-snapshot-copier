package rdsops

import (
	"log"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/bluebenno/rds-snapshot-copier/internal/wiring"
	"go.uber.org/zap"
)

func TestList(t *testing.T) {
	t.Parallel()
	i01 := rds.DBInstance{
		DBInstanceIdentifier: aws.String("instance-i01"),
		DBInstanceArn:        aws.String("dummyarn1"),
	}
	i02 := rds.DBInstance{
		DBInstanceIdentifier: aws.String("instance-i02"),
		DBInstanceArn:        aws.String("dummyarn2"),
	}
	var marker string

	type want struct {
		err    bool
		result []*rds.DBInstance
	}
	tests := []struct {
		awsmockresult *rds.DescribeDBInstancesOutput
		name          string
		want          want
	}{
		{
			name: "List_Simple01",
			want: want{
				err:    false,
				result: []*rds.DBInstance{&i01},
			},
			awsmockresult: &rds.DescribeDBInstancesOutput{
				DBInstances: []*rds.DBInstance{&i01},
				Marker:      &marker,
			},
		},
		{
			name: "List_Simple0102",
			want: want{
				err:    false,
				result: []*rds.DBInstance{&i01, &i02},
			},
			awsmockresult: &rds.DescribeDBInstancesOutput{
				DBInstances: []*rds.DBInstance{&i01, &i02},
			},
		},
		{
			name: "List_Simple_nil",
			want: want{
				err:    false,
				result: nil,
			},
			awsmockresult: &rds.DescribeDBInstancesOutput{
				DBInstances: []*rds.DBInstance{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				describeDBInstancesOutput: tt.awsmockresult,
			}

			got, err := List(mockSvc)
			if (err != nil) != tt.want.err {
				t.Errorf("List() error = %v, wantErr %v", err, tt.want.err)
				return
			}

			if !reflect.DeepEqual(got, tt.want.result) {
				t.Errorf("%v = %v, want %v", tt.name, got, tt.want.result)
			}

		})
	}
}

func TestGetTag(t *testing.T) {
	t.Parallel()

	type args struct {
		arn    string
		search string
	}
	type want struct {
		err    bool
		result string
	}
	tests := []struct {
		args          args
		awsmockresult *rds.ListTagsForResourceOutput
		name          string
		want          want
	}{
		{
			name: "GetTag_foundfoo",
			args: args{
				arn:    "arn:aws:rds:ap-southeast-2:200000000000:db:dummyvalue",
				search: "foo",
			},
			want: want{
				result: "bar",
				err:    false,
			},
			awsmockresult: &rds.ListTagsForResourceOutput{
				TagList: []*rds.Tag{
					{Key: aws.String("foo1"), Value: aws.String("bar1")},
					{Key: aws.String("foo"), Value: aws.String("bar")},
					{Key: aws.String("foo2"), Value: aws.String("bar2")},
				},
			},
		},
		{
			name: "GetTag_notfound",
			args: args{
				arn:    "arn:aws:rds:ap-southeast-2:200000000000:db:dummyvalue",
				search: "searchtermnotfound",
			},
			want: want{
				result: "",
				err:    false,
			},
			awsmockresult: &rds.ListTagsForResourceOutput{
				TagList: []*rds.Tag{
					{Key: aws.String("foo"), Value: aws.String("bar")},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				listTagsForResourceOutput: tt.awsmockresult,
			}

			got, err := GetTag(mockSvc, tt.args.arn, tt.args.search)

			if (err != nil) != tt.want.err {
				t.Errorf("List() error = %v, wantErr %v", err, tt.want.err)
				return
			}

			if !reflect.DeepEqual(got, tt.want.result) {
				t.Errorf("%v = %v, want %v", tt.name, got, tt.want.result)
			}

		})
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()
	i01 := rds.DBInstance{
		DBInstanceIdentifier: aws.String("instance-i01"),
		DBInstanceStatus:     aws.String("available"),
		DBInstanceArn:        aws.String("dummyarn1"),
	}
	i02 := rds.DBInstance{
		DBInstanceIdentifier: aws.String("instance-i02"),
		DBInstanceStatus:     aws.String("notavailable"),
		DBInstanceArn:        aws.String("dummyarn2"),
	}
	instances01 := []*rds.DBInstance{&i01}
	instances02 := []*rds.DBInstance{&i02}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to create logger: %s", err.Error())
	}

	var cfg wiring.Config

	type args struct {
		logger *zap.Logger
		input  []*rds.DBInstance
		cfg    *wiring.Config
		tag    string
	}
	type want struct {
		err    bool
		result []*rds.DBInstance
	}
	tests := []struct {
		args          args
		awsmockresult *rds.ListTagsForResourceOutput
		name          string
		want          want
	}{
		{
			name: "Filter_found-i01-available",
			args: args{
				logger: logger,
				input:  instances01,
				tag:    "copythisone",
			},
			want: want{
				result: instances01,
				err:    false,
			},
			awsmockresult: &rds.ListTagsForResourceOutput{
				TagList: []*rds.Tag{
					{Key: aws.String("copythisone"), Value: aws.String("anyvalue")},
				},
			},
		},
		{
			name: "Filter_found-i01-nottagged",
			args: args{
				logger: logger,
				input:  instances01,
			},
			want: want{
				result: nil,
				err:    false,
			},
			awsmockresult: &rds.ListTagsForResourceOutput{
				TagList: []*rds.Tag{
					{Key: aws.String(""), Value: aws.String("")},
				},
			},
		},
		{
			name: "Filter_found-i02-not-available",
			args: args{
				logger: logger,
				input:  instances02,
				tag:    "copythisone",
			},
			want: want{
				result: nil,
				err:    false,
			},
			awsmockresult: &rds.ListTagsForResourceOutput{
				TagList: []*rds.Tag{
					{Key: aws.String("copythisone"), Value: aws.String("anyvalue")},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				listTagsForResourceOutput: tt.awsmockresult,
			}
			cfg.Tag = tt.args.tag
			got, err := Filter(tt.args.logger, &cfg, mockSvc, tt.args.input)

			if (err != nil) != tt.want.err {
				t.Errorf("List() error = %v, wantErr %v", err, tt.want.err)
				return
			}

			if !reflect.DeepEqual(got, tt.want.result) {
				t.Errorf("%v = %v, want %v", tt.name, got, tt.want.result)
			}

		})
	}
}

// Defines a mock struct to be used for unit tests
type mockRDSClient struct {
	rdsiface.RDSAPI
	describeDBInstancesOutput *rds.DescribeDBInstancesOutput
	listTagsForResourceOutput *rds.ListTagsForResourceOutput
}

// Mock DescribeDBInstances
func (m *mockRDSClient) DescribeDBInstances(i *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	return m.describeDBInstancesOutput, nil
}

// Mock ListTagsForResource
func (m *mockRDSClient) ListTagsForResource(i *rds.ListTagsForResourceInput) (*rds.ListTagsForResourceOutput, error) {
	return m.listTagsForResourceOutput, nil
}

// Mock DescribeDBInstancesPages
func (m *mockRDSClient) DescribeDBInstancesPages(i *rds.DescribeDBInstancesInput, fn func(*rds.DescribeDBInstancesOutput, bool) bool) error {
	fn(m.describeDBInstancesOutput, true)
	return nil
}
