package main

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapprunner"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecrassets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsmemorydb"
	"github.com/aws/jsii-runtime-go"

	"github.com/aws/constructs-go/constructs/v10"
)

type CdkStackProps struct {
	awscdk.StackProps
}

func main() {
	app := awscdk.NewApp(nil)

	NewCdkStack1(app, "MemoryDBAppRunnerInfraStack", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	NewCdkStack2(app, "MemoryDBAppRunnerAppStack1", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

const (
	memoryDBNodeType                  = "db.t4g.small"
	accessString                      = "on ~* &* +@all"
	numMemoryDBShards                 = 1
	numMemoryDBReplicaPerShard        = 1
	memoryDBDefaultParameterGroupName = "default.memorydb-redis6"
	memoryDBRedisEngineVersion        = "6.2"
	memoryDBRedisPort                 = 6379

	appRunnerServicePrincipal             = "build.apprunner.amazonaws.com"
	appRunnerEgressType                   = "VPC"
	appRunnerServicePolicyForECRAccessARN = "arn:aws:iam::aws:policy/service-role/AWSAppRunnerServicePolicyForECRAccess"

	ecrImageRepositoryType = "ECR"
)

var memorydbCluster awsmemorydb.CfnCluster
var vpc awsec2.Vpc
var user awsmemorydb.CfnUser
var appRunnerVPCConnSecurityGroup awsec2.SecurityGroup

func NewCdkStack1(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc = awsec2.NewVpc(stack, jsii.String("demo-vpc"), nil)

	authInfo := map[string]interface{}{"Type": "password", "Passwords": []string{getMemoryDBPassword()}}

	user = awsmemorydb.NewCfnUser(stack, jsii.String("demo-memorydb-user"), &awsmemorydb.CfnUserProps{UserName: jsii.String("demo-user"), AccessString: jsii.String(accessString), AuthenticationMode: authInfo})

	acl := awsmemorydb.NewCfnACL(stack, jsii.String("demo-memorydb-acl"), &awsmemorydb.CfnACLProps{AclName: jsii.String("demo-memorydb-acl"), UserNames: &[]*string{user.UserName()}})

	acl.AddDependsOn(user)

	var subnetIDsForSubnetGroup []*string

	for _, sn := range *vpc.PrivateSubnets() {
		subnetIDsForSubnetGroup = append(subnetIDsForSubnetGroup, sn.SubnetId())
	}

	subnetGroup := awsmemorydb.NewCfnSubnetGroup(stack, jsii.String("demo-memorydb-subnetgroup"), &awsmemorydb.CfnSubnetGroupProps{SubnetGroupName: jsii.String("demo-memorydb-subnetgroup"), SubnetIds: &subnetIDsForSubnetGroup})

	memorydbSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("memorydb-demo-sg"), &awsec2.SecurityGroupProps{Vpc: vpc, SecurityGroupName: jsii.String("memorydb-demo-sg"), AllowAllOutbound: jsii.Bool(true)})

	memorydbCluster = awsmemorydb.NewCfnCluster(stack, jsii.String("demo-memorydb-cluster"), &awsmemorydb.CfnClusterProps{ClusterName: jsii.String("demo-memorydb-cluster"), NodeType: jsii.String(memoryDBNodeType), AclName: acl.AclName(), NumShards: jsii.Number(numMemoryDBShards), EngineVersion: jsii.String(memoryDBRedisEngineVersion), Port: jsii.Number(memoryDBRedisPort), SubnetGroupName: subnetGroup.SubnetGroupName(), NumReplicasPerShard: jsii.Number(numMemoryDBReplicaPerShard), TlsEnabled: jsii.Bool(true), SecurityGroupIds: &[]*string{memorydbSecurityGroup.SecurityGroupId()}, ParameterGroupName: jsii.String(memoryDBDefaultParameterGroupName)})

	memorydbCluster.AddDependsOn(user)
	memorydbCluster.AddDependsOn(acl)
	memorydbCluster.AddDependsOn(subnetGroup)

	appRunnerVPCConnSecurityGroup = awsec2.NewSecurityGroup(stack, jsii.String("apprunner-demo-sg"), &awsec2.SecurityGroupProps{Vpc: vpc, SecurityGroupName: jsii.String("apprunner-demo-sg"), AllowAllOutbound: jsii.Bool(true)})

	memorydbSecurityGroup.AddIngressRule(awsec2.Peer_SecurityGroupId(appRunnerVPCConnSecurityGroup.SecurityGroupId(), nil), awsec2.Port_Tcp(jsii.Number(memoryDBRedisPort)), jsii.String("for apprunner to access memorydb"), jsii.Bool(false))

	return stack
}

func NewCdkStack2(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	ecrAccessPolicy := awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("ecr-access-policy"), jsii.String(appRunnerServicePolicyForECRAccessARN))

	apprunnerECRIAMrole := awsiam.NewRole(stack, jsii.String("role-apprunner-ecr"), &awsiam.RoleProps{AssumedBy: awsiam.NewServicePrincipal(jsii.String(appRunnerServicePrincipal), nil), RoleName: jsii.String("role-apprunner-ecr"), ManagedPolicies: &[]awsiam.IManagedPolicy{ecrAccessPolicy}})

	ecrAccessRoleConfig := awsapprunner.CfnService_AuthenticationConfigurationProperty{AccessRoleArn: apprunnerECRIAMrole.RoleArn()}

	memoryDBEndpointURL := fmt.Sprintf("%s:%s", *memorydbCluster.AttrClusterEndpointAddress(), strconv.Itoa(int(*memorydbCluster.Port())))

	appRunnerServiceEnvVarConfig := []awsapprunner.CfnService_KeyValuePairProperty{{Name: jsii.String("MEMORYDB_CLUSTER_ENDPOINT"), Value: jsii.String(memoryDBEndpointURL)}, {Name: jsii.String("MEMORYDB_USERNAME"), Value: user.UserName()}, {Name: jsii.String("MEMORYDB_PASSWORD"), Value: jsii.String(getMemoryDBPassword())}}

	imageConfig := awsapprunner.CfnService_ImageConfigurationProperty{RuntimeEnvironmentVariables: appRunnerServiceEnvVarConfig, Port: jsii.String(getAppRunnerServicePort())}

	//--------BUILD DOCKER IMAGE---------
	appDockerImage := awsecrassets.NewDockerImageAsset(stack, jsii.String("app-image"), &awsecrassets.DockerImageAssetProps{Directory: jsii.String("../app/")})

	sourceConfig := awsapprunner.CfnService_SourceConfigurationProperty{AuthenticationConfiguration: ecrAccessRoleConfig, ImageRepository: awsapprunner.CfnService_ImageRepositoryProperty{ImageIdentifier: jsii.String(*appDockerImage.ImageUri()), ImageRepositoryType: jsii.String(ecrImageRepositoryType), ImageConfiguration: imageConfig}}

	var subnetIDsForSubnetGroup []*string

	for _, sn := range *vpc.PrivateSubnets() {
		subnetIDsForSubnetGroup = append(subnetIDsForSubnetGroup, sn.SubnetId())
	}

	vpcConnector := awsapprunner.NewCfnVpcConnector(stack, jsii.String("apprunner-vpc-connector"), &awsapprunner.CfnVpcConnectorProps{Subnets: &subnetIDsForSubnetGroup, SecurityGroups: &[]*string{appRunnerVPCConnSecurityGroup.SecurityGroupId()}, VpcConnectorName: jsii.String("demo-apprunner-vpc-connector")})

	networkConfig := awsapprunner.CfnService_NetworkConfigurationProperty{EgressConfiguration: awsapprunner.CfnService_EgressConfigurationProperty{EgressType: jsii.String(appRunnerEgressType), VpcConnectorArn: vpcConnector.AttrVpcConnectorArn()}}

	app := awsapprunner.NewCfnService(stack, jsii.String("apprunner-app"), &awsapprunner.CfnServiceProps{SourceConfiguration: sourceConfig, ServiceName: jsii.String(getAppRunnerServiceName()), NetworkConfiguration: networkConfig})

	app.AddDependsOn(vpcConnector)

	//app.CfnOptions().SetDeletionPolicy(awscdk.CfnDeletionPolicy_RETAIN)

	return stack

}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	return nil
}
