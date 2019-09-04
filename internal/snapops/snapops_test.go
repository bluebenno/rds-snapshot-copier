package snapops

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"

	"github.com/bluebenno/RDS-snapshot-copier/internal/wiring"
)

func TestGetSlice(t *testing.T) {
	t.Parallel()
	s1 := rds.DBSnapshot{DBSnapshotIdentifier: aws.String("one")}
	s2 := rds.DBSnapshot{DBSnapshotIdentifier: aws.String("two")}
	s3 := rds.DBSnapshot{DBSnapshotIdentifier: aws.String("three")}

	type args struct {
		all   []*rds.DBSnapshot
		start int
		num   int
	}
	type want struct {
		result []*rds.DBSnapshot
		err    bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "GetSlice_-1,2",
			args: args{
				all:   []*rds.DBSnapshot{&s1, &s2, &s3},
				start: -1,
				num:   2,
			},
			want: want{
				result: []*rds.DBSnapshot{&s2, &s3},
				err:    false,
			},
		},
		{
			name: "GetSlice_-1,1",
			args: args{
				all:   []*rds.DBSnapshot{&s1, &s2, &s3},
				start: -1,
				num:   1,
			},
			want: want{
				result: []*rds.DBSnapshot{&s3},
				err:    false,
			},
		},
		{
			name: "GetSlice_0,1",
			args: args{
				all:   []*rds.DBSnapshot{&s1, &s2, &s3},
				start: 0,
				num:   1,
			},
			want: want{
				result: []*rds.DBSnapshot{&s1},
				err:    false,
			},
		},
		{
			name: "GetSlice_1,1",
			args: args{
				all:   []*rds.DBSnapshot{&s1, &s2, &s3},
				start: 1,
				num:   1,
			},
			want: want{
				result: []*rds.DBSnapshot{&s2},
				err:    false,
			},
		},
		{
			name: "GetSlice_0,3",
			args: args{
				all:   []*rds.DBSnapshot{&s1, &s2, &s3},
				start: 0,
				num:   3,
			},
			want: want{
				result: []*rds.DBSnapshot{&s1, &s2, &s3},
				err:    false,
			},
		},
		{
			name: "GetSlice_null",
			args: args{
				all:   nil,
				start: 1,
				num:   1,
			},
			want: want{
				result: nil,
				err:    false,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSlice(tt.args.all, tt.args.start, tt.args.num)

			if (err != nil) != tt.want.err {
				t.Errorf("GetSlice() error = %v, wantErr %v", err, tt.want.err)
				return
			}

			if !reflect.DeepEqual(got, tt.want.result) {
				t.Errorf("%v = %v, want %v", tt.name, got, tt.want.result)
			}

		})
	}
}

func TestGetLatest(t *testing.T) {
	t.Parallel()
	s1 := rds.DBSnapshot{DBSnapshotIdentifier: aws.String("one")}
	s2 := rds.DBSnapshot{DBSnapshotIdentifier: aws.String("two")}

	type args struct {
		all   []*rds.DBSnapshot
		start int
		num   int
	}
	type want struct {
		result *rds.DBSnapshot
		err    bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "getlatest_simple",
			args: args{
				all:   []*rds.DBSnapshot{&s1, &s2},
				start: -1,
				num:   1,
			},
			want: want{
				result: &s2,
				err:    false,
			},
		},
		{
			name: "getlatest_empty",
			args: args{
				all:   []*rds.DBSnapshot{},
				start: -1,
				num:   1,
			},
			want: want{
				result: nil,
				err:    false,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLatest(tt.args.all)

			if (err != nil) != tt.want.err {
				t.Errorf("GetSlice() error = %v, wantErr %v", err, tt.want.err)
				return
			}

			if !reflect.DeepEqual(got, tt.want.result) {
				t.Errorf("%v = %v, want %v", tt.name, got, tt.want.result)
			}

		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	i1s1 := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap01"),
	}

	type args struct {
		dbinstance string
	}
	type want struct {
		err    bool
		result []*rds.DBSnapshot
	}
	tests := []struct {
		args          args
		awsmockresult *rds.DescribeDBSnapshotsOutput
		name          string
		want          want
	}{
		{
			name: "getsnapshots_i1s1",
			args: args{
				dbinstance: "dbinstance-one",
			},
			want: want{
				result: []*rds.DBSnapshot{&i1s1},
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{
				DBSnapshots: []*rds.DBSnapshot{&i1s1},
			},
		},
		{
			name: "getsnapshots_nonefound",
			args: args{
				dbinstance: "itwillnotbefound",
			},
			want: want{
				result: nil,
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				describeDBSnapShotOutput: tt.awsmockresult,
			}

			got, err := List(mockSvc, tt.args.dbinstance)

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

func TestPullSnapShot(t *testing.T) {
	t.Parallel()
	sres01 := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap01"),
	}

	type args struct {
		config             wiring.Config
		arn                string
		targetsnapshotname string
	}
	type want struct {
		err    bool
		result *rds.CopyDBSnapshotOutput
	}
	tests := []struct {
		args          args
		awsmockresult *rds.CopyDBSnapshotOutput
		name          string
		want          want
	}{
		{
			name: "pullsnapshot_simple_happy",
			args: args{
				arn:                "dbinstance-one",
				targetsnapshotname: "target",
			},
			want: want{
				result: &rds.CopyDBSnapshotOutput{
					DBSnapshot: &sres01,
				},
				err: false,
			},
			awsmockresult: &rds.CopyDBSnapshotOutput{
				DBSnapshot: &sres01,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				copyDBSnapshotOutput: tt.awsmockresult,
			}

			got, err := PullSnapShot(&tt.args.config, mockSvc, &tt.args.arn, tt.args.targetsnapshotname)

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

func TestListExpired(t *testing.T) {
	t.Parallel()
	tone, _ := time.Parse(time.RFC822, "01 Jan 11 01:00 AEST")
	ttwo, _ := time.Parse(time.RFC822, "02 Jan 12 02:00 AEST")
	tthree, _ := time.Parse(time.RFC822, "03 Jan 13 03:00 AEST")

	sres01 := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap01"),
		SnapshotCreateTime:   &tone,
	}
	sres02 := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap02"),
		SnapshotCreateTime:   &ttwo,
	}
	sres03 := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap03"),
		SnapshotCreateTime:   &tthree,
	}

	type args struct {
		config   wiring.Config
		instance *rds.DBInstance
	}

	type want struct {
		err    bool
		result []*rds.DBSnapshot
	}
	tests := []struct {
		args          args
		awsmockresult *rds.DescribeDBSnapshotsOutput
		name          string
		want          want
	}{
		{
			name: "ListExpired_retain1_expire2",
			args: args{
				config: wiring.Config{
					MaxSnap: 1,
				},
				instance: &rds.DBInstance{
					DBInstanceIdentifier: aws.String("one"),
				},
			},
			want: want{
				result: []*rds.DBSnapshot{&sres01, &sres02},
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{
				DBSnapshots: []*rds.DBSnapshot{&sres01, &sres02, &sres03},
			},
		},
		{
			name: "ListExpired_retain2_expire1",
			args: args{
				config: wiring.Config{
					MaxSnap: 2,
				},
				instance: &rds.DBInstance{
					DBInstanceIdentifier: aws.String("one"),
				},
			},
			want: want{
				result: []*rds.DBSnapshot{&sres01},
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{
				DBSnapshots: []*rds.DBSnapshot{&sres01, &sres02, &sres03},
			},
		},
		{
			name: "ListExpired_retain3_expire0-i",
			args: args{
				config: wiring.Config{
					MaxSnap: 3,
				},
				instance: &rds.DBInstance{
					DBInstanceIdentifier: aws.String("one"),
				},
			},
			want: want{
				result: nil,
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{
				DBSnapshots: []*rds.DBSnapshot{&sres01, &sres02, &sres03},
			},
		},
		{
			name: "ListExpired_retain3_expire0-ii",
			args: args{
				config: wiring.Config{
					MaxSnap: 4,
				},
				instance: &rds.DBInstance{
					DBInstanceIdentifier: aws.String("one"),
				},
			},
			want: want{
				result: nil,
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{
				DBSnapshots: []*rds.DBSnapshot{&sres01, &sres02, &sres03},
			},
		},
		{
			name: "ListExpired_retain0_expire3",
			args: args{
				config: wiring.Config{
					MaxSnap: 0,
				},
				instance: &rds.DBInstance{
					DBInstanceIdentifier: aws.String("one"),
				},
			},
			want: want{
				result: []*rds.DBSnapshot{&sres01, &sres02, &sres03},
				err:    false,
			},
			awsmockresult: &rds.DescribeDBSnapshotsOutput{
				DBSnapshots: []*rds.DBSnapshot{&sres01, &sres02, &sres03},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				describeDBSnapShotOutput: tt.awsmockresult,
			}

			got, err := ListExpired(&tt.args.config, mockSvc, tt.args.instance)

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

func TestDelete(t *testing.T) {
	t.Parallel()
	sres01 := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap01"),
	}
	sres01del := rds.DBSnapshot{
		DBInstanceIdentifier: aws.String("dbinstance-one"),
		DBSnapshotIdentifier: aws.String("dbinstance-one-snap01"),
		Status:               aws.String("deleted"),
	}
	type args struct {
		snaps []*rds.DBSnapshot
	}
	type want struct {
		err    bool
		result int
	}
	tests := []struct {
		args          args
		awsmockresult *rds.DeleteDBSnapshotOutput
		name          string
		want          want
	}{
		{
			name: "TestDelete_empty",
			args: args{
				snaps: []*rds.DBSnapshot{},
			},
			want: want{
				result: 0,
				err:    false,
			},
		},
		{
			name: "TestDelete_nil",
			args: args{
				snaps: nil,
			},
			want: want{
				result: 0,
				err:    false,
			},
		},
		{
			name: "TestDelete_one",
			args: args{
				snaps: []*rds.DBSnapshot{&sres01},
			},
			want: want{
				result: 1,
				err:    false,
			},
			awsmockresult: &rds.DeleteDBSnapshotOutput{
				DBSnapshot: &sres01del,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockRDSClient{
				deleteDBSnapshotOutput: tt.awsmockresult,
			}

			got, err := Delete(mockSvc, tt.args.snaps)

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
	describeDBSnapShotOutput *rds.DescribeDBSnapshotsOutput
	copyDBSnapshotOutput     *rds.CopyDBSnapshotOutput
	deleteDBSnapshotOutput   *rds.DeleteDBSnapshotOutput
}

// Mock CopyDBSnapshot
func (m *mockRDSClient) CopyDBSnapshot(i *rds.CopyDBSnapshotInput) (*rds.CopyDBSnapshotOutput, error) {
	return m.copyDBSnapshotOutput, nil
}

// Mock DescribeDBSnapshots
func (m *mockRDSClient) DescribeDBSnapshots(i *rds.DescribeDBSnapshotsInput) (*rds.DescribeDBSnapshotsOutput, error) {
	return m.describeDBSnapShotOutput, nil
}

// Mock DeleteDBSnapshot
func (m *mockRDSClient) DeleteDBSnapshot(i *rds.DeleteDBSnapshotInput) (*rds.DeleteDBSnapshotOutput, error) {
	return m.deleteDBSnapshotOutput, nil
}

// Mock DescribeDBSnapshotPages
func (m *mockRDSClient) DescribeDBSnapshotsPages(i *rds.DescribeDBSnapshotsInput, fn func(*rds.DescribeDBSnapshotsOutput, bool) bool) error {
	fn(m.describeDBSnapShotOutput, true)
	return nil
}
