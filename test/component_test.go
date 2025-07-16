package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	awshelper "github.com/cloudposse/test-helpers/pkg/aws"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
)

type ComponentSuite struct {
	helper.TestSuite
}

var (
	suffix   = strings.ToLower(random.UniqueId())
	userName = "test_user"
	password = random.UniqueId() + random.UniqueId()[:4] // Combine two IDs to ensure at least 8 characters
)

func (s *ComponentSuite) TestBasic() {
	const component = "documentdb/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	inputs := map[string]any{
		"master_username": userName,
		"master_password": password,
		"name":            fmt.Sprintf("%s-docdb", suffix),
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	masterUsername := atmos.Output(s.T(), options, "master_username")
	assert.Equal(s.T(), userName, masterUsername)

	clusterName := atmos.Output(s.T(), options, "cluster_name")
	assert.NotEmpty(s.T(), clusterName)

	arn := atmos.Output(s.T(), options, "arn")
	assert.NotEmpty(s.T(), arn)

	endpoint := atmos.Output(s.T(), options, "endpoint")
	assert.NotEmpty(s.T(), endpoint)

	readerEndpoint := atmos.Output(s.T(), options, "reader_endpoint")
	assert.NotEmpty(s.T(), readerEndpoint)

	masterEndpoint := atmos.Output(s.T(), options, "master_host")
	assert.NotEmpty(s.T(), masterEndpoint)

	replicasHost := atmos.Output(s.T(), options, "replicas_host")
	assert.NotEmpty(s.T(), replicasHost)

	securityGroupID := atmos.Output(s.T(), options, "security_group_id")
	assert.NotEmpty(s.T(), securityGroupID)

	securityGroupArn := atmos.Output(s.T(), options, "security_group_arn")
	assert.NotEmpty(s.T(), securityGroupArn)

	securityGroupName := atmos.Output(s.T(), options, "security_group_name")
	assert.NotEmpty(s.T(), securityGroupName)

	client := awshelper.NewDocDBClient(s.T(), awsRegion)
	clusters, err := client.DescribeDBClusters(context.Background(), &docdb.DescribeDBClustersInput{
		DBClusterIdentifier: &clusterName,
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), arn, *clusters.DBClusters[0].DBClusterArn)
	assert.Equal(s.T(), clusterName, *clusters.DBClusters[0].DBClusterIdentifier)
	assert.Equal(s.T(), readerEndpoint, *clusters.DBClusters[0].ReaderEndpoint)
	assert.Equal(s.T(), securityGroupID, *clusters.DBClusters[0].VpcSecurityGroups[0].VpcSecurityGroupId)

	dnsDelegatedOptions := s.GetAtmosOptions("dns-delegated", "default-test", nil)
	delegatedDNSZoneID := atmos.Output(s.T(), dnsDelegatedOptions, "default_dns_zone_id")
	masterEndpointDNSRecord := aws.GetRoute53Record(s.T(), delegatedDNSZoneID, masterEndpoint, "CNAME", awsRegion)
	assert.Equal(s.T(), *masterEndpointDNSRecord.ResourceRecords[0].Value, *clusters.DBClusters[0].Endpoint)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "documentdb/disabled"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	inputs := map[string]any{
		"master_username": userName,
		"master_password": password,
		"name":            fmt.Sprintf("%s-docdb", suffix),
	}

	s.VerifyEnabledFlag(component, stack, &inputs)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)

	suite.AddDependency(t, "vpc", "default-test", nil)

	subdomain := strings.ToLower(random.UniqueId())
	inputs := map[string]any{
		"zone_config": []map[string]any{
			{
				"subdomain": subdomain,
				"zone_name": "components.cptest.test-automation.app",
			},
		},
	}
	suite.AddDependency(t, "dns-delegated", "default-test", &inputs)
	helper.Run(t, suite)
}
