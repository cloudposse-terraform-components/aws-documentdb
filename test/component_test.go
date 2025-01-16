package test

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/aws-component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent(t *testing.T) {
	// Define the AWS region to use for the tests
	awsRegion := "us-east-2"

	// Initialize the test fixture
	fixture := helper.NewFixture(t, "../", awsRegion, "test/fixtures")

	// Ensure teardown is executed after the test
	defer fixture.TearDown()
	fixture.SetUp(&atmos.Options{})

	// Define the test suite
	fixture.Suite("default", func(t *testing.T, suite *helper.Suite) {
		suite.AddDependency("vpc", "default-test")

		// Setup phase: Create DNS zones for testing
		suite.Setup(t, func(t *testing.T, atm *helper.Atmos) {
			basicDomain := "components.cptest.test-automation.app"

			// Deploy the delegated DNS zone
			inputs := map[string]interface{}{
				"zone_config": []map[string]interface{}{
					{
						"subdomain": suite.GetRandomIdentifier(),
						"zone_name": basicDomain,
					},
				},
			}
			atm.GetAndDeploy("dns-delegated", "default-test", inputs)
		})

		// Teardown phase: Destroy the DNS zones created during setup
		suite.TearDown(t, func(t *testing.T, atm *helper.Atmos) {
			// Deploy the delegated DNS zone
			inputs := map[string]interface{}{
				"zone_config": []map[string]interface{}{
					{
						"subdomain": suite.GetRandomIdentifier(),
						"zone_name": "components.cptest.test-automation.app",
					},
				},
			}
			atm.GetAndDestroy("dns-delegated", "default-test", inputs)
		})

		// Test phase: Validate the functionality of the ALB component
		suite.Test(t, "basic", func(t *testing.T, atm *helper.Atmos) {
			name := strings.ToLower(random.UniqueId())
			userName := strings.ToLower(random.UniqueId())
			inputs := map[string]interface{}{
				"master_username": userName,
				"name":            name,
			}

			defer atm.GetAndDestroy("documentdb/basic", "default-test", inputs)
			component := atm.GetAndDeploy("documentdb/basic", "default-test", inputs)
			assert.NotNil(t, component)

			masterUsername := atm.Output(component, "master_username")
			assert.Equal(t, userName, masterUsername)

			clusterName := atm.Output(component, "cluster_name")
			assert.NotEmpty(t, clusterName)

			arn := atm.Output(component, "arn")
			assert.NotEmpty(t, arn)

			endpoint := atm.Output(component, "endpoint")
			assert.NotEmpty(t, endpoint)

			readerEndpoint := atm.Output(component, "reader_endpoint")
			assert.NotEmpty(t, readerEndpoint)

			masterEndpoint := atm.Output(component, "master_host")
			assert.NotEmpty(t, masterEndpoint)

			replicasHost := atm.Output(component, "replicas_host")
			assert.NotEmpty(t, replicasHost)

			securityGroupId := atm.Output(component, "security_group_id")
			assert.NotEmpty(t, securityGroupId)

			securityGroupArn := atm.Output(component, "security_group_arn")
			assert.NotEmpty(t, securityGroupArn)

			securityGroupName := atm.Output(component, "security_group_name")
			assert.NotEmpty(t, securityGroupName)

			client := NewDocDBlient(t, awsRegion)
			clusters, err := client.DescribeDBClusters(context.Background(), &docdb.DescribeDBClustersInput{
				DBClusterIdentifier: &clusterName,
			})
			assert.NoError(t, err)
			assert.Equal(t, arn, *clusters.DBClusters[0].DBClusterArn)
			assert.Equal(t, clusterName, *clusters.DBClusters[0].DBClusterIdentifier)
			assert.Equal(t, readerEndpoint, *clusters.DBClusters[0].ReaderEndpoint)
			assert.Equal(t, securityGroupId, *clusters.DBClusters[0].VpcSecurityGroups[0].VpcSecurityGroupId)

			delegatedDnsZoneId := atm.Output(helper.NewAtmosComponent("dns-delegated", "default-test", map[string]interface{}{}), "default_dns_zone_id")
			masterEndpointDNSRecord := aws.GetRoute53Record(t, delegatedDnsZoneId, masterEndpoint, "CNAME", awsRegion)
			assert.Equal(t, *masterEndpointDNSRecord.ResourceRecords[0].Value, *clusters.DBClusters[0].Endpoint)
		})
	})
}

func NewDocDBlient(t *testing.T, region string) *docdb.Client {
	client, err := NewDocDBlientE(t, region)
	require.NoError(t, err)

	return client
}

func NewDocDBlientE(t *testing.T, region string) (*docdb.Client, error) {
	sess, err := aws.NewAuthenticatedSession(region)
	if err != nil {
		return nil, err
	}
	return docdb.NewFromConfig(*sess), nil
}
