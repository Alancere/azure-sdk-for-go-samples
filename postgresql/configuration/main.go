package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	subscriptionID    string
	location          = "eastus"
	resourceGroupName = "sample-resource-group"
	serverName        = "sampleyserver"
	configurationName = "sample-postgresql-configuration"
)

func main() {
	subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	resourceGroup, err := createResourceGroup(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("resources group:", *resourceGroup.ID)

	server, err := createServer(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("postgresql server:", *server.ID)

	configuration, err := createConfiguration(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("postgresql configuration:", *configuration.ID)

	keepResource := os.Getenv("KEEP_RESOURCE")
	if len(keepResource) == 0 {
		_, err := cleanup(ctx, cred)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("cleaned up successfully.")
	}
}

func createServer(ctx context.Context, cred azcore.TokenCredential) (*armpostgresql.Server, error) {
	serversClient := armpostgresql.NewServersClient(subscriptionID, cred, nil)

	pollerResp, err := serversClient.BeginCreate(
		ctx,
		resourceGroupName,
		serverName,
		armpostgresql.ServerForCreate{
			Location: to.StringPtr(location),
			Properties: &armpostgresql.ServerPropertiesForDefaultCreate{
				ServerPropertiesForCreate: armpostgresql.ServerPropertiesForCreate{
					CreateMode:               armpostgresql.CreateModeDefault.ToPtr(),
					InfrastructureEncryption: armpostgresql.InfrastructureEncryptionDisabled.ToPtr(),
					PublicNetworkAccess:      armpostgresql.PublicNetworkAccessEnumEnabled.ToPtr(),
					Version:                  armpostgresql.ServerVersionEleven.ToPtr(),
				},
				AdministratorLogin:         to.StringPtr("dummylogin"),
				AdministratorLoginPassword: to.StringPtr("QWE123!@#"),
			},
			SKU: &armpostgresql.SKU{
				Name: to.StringPtr("B_Gen5_1"),
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

func createConfiguration(ctx context.Context, cred azcore.TokenCredential) (*armpostgresql.Configuration, error) {
	configurationsClient := armpostgresql.NewConfigurationsClient(subscriptionID, cred, nil)

	pollerResp, err := configurationsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		serverName,
		configurationName,
		armpostgresql.Configuration{
			Properties: &armpostgresql.ConfigurationProperties{
				Source: to.StringPtr("user-override"),
				Value:  to.StringPtr("off"),
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.Configuration, nil
}

func createResourceGroup(ctx context.Context, cred azcore.TokenCredential) (*armresources.ResourceGroup, error) {
	resourceGroupClient := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	resourceGroupResp, err := resourceGroupClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		armresources.ResourceGroup{
			Location: to.StringPtr(location),
		},
		nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
}

func cleanup(ctx context.Context, cred azcore.TokenCredential) (*http.Response, error) {
	resourceGroupClient := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return resp.RawResponse, nil
}
