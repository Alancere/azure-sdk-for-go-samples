package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	subscriptionID    string
	location          = "westus"
	resourceGroupName = "sample-resource-group"
	registryName      = "sample2registry"
	taskRunName       = "sample-task-run"
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

	conn := arm.NewDefaultConnection(cred, &arm.ConnectionOptions{
		Logging: policy.LogOptions{
			IncludeBody: true,
		},
	})
	ctx := context.Background()

	resourceGroup, err := createResourceGroup(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("resources group:", *resourceGroup.ID)

	registry, err := createRegistry(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("registry:", *registry.ID)

	taskRun, err := createTaskRun(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("task run:", *taskRun.ID)

	taskRun, err = getTaskRun(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("get task run:", *taskRun.ID)

	keepResource := os.Getenv("KEEP_RESOURCE")
	if len(keepResource) == 0 {
		_, err := cleanup(ctx, conn)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("cleaned up successfully.")
	}
}

func createRegistry(ctx context.Context, conn *arm.Connection) (*armcontainerregistry.Registry, error) {
	registriesClient := armcontainerregistry.NewRegistriesClient(conn, subscriptionID)

	pollerResp, err := registriesClient.BeginCreate(
		ctx,
		resourceGroupName,
		registryName,
		armcontainerregistry.Registry{
			Resource: armcontainerregistry.Resource{
				Location: to.StringPtr(location),
				Tags: map[string]*string{
					"key": to.StringPtr("value"),
				},
			},
			SKU: &armcontainerregistry.SKU{
				Name: armcontainerregistry.SKUNamePremium.ToPtr(),
			},
			Properties: &armcontainerregistry.RegistryProperties{
				AdminUserEnabled: to.BoolPtr(true),
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
	return &resp.Registry, nil
}

func createTaskRun(ctx context.Context, conn *arm.Connection) (*armcontainerregistry.TaskRun, error) {
	taskRunsClient := armcontainerregistry.NewTaskRunsClient(conn, subscriptionID)

	pollerResp, err := taskRunsClient.BeginCreate(
		ctx,
		resourceGroupName,
		registryName,
		taskRunName,
		armcontainerregistry.TaskRun{
			Properties: &armcontainerregistry.TaskRunProperties{
				ForceUpdateTag: to.StringPtr("test"),
				RunRequest: &armcontainerregistry.DockerBuildRequest{
					RunRequest: armcontainerregistry.RunRequest{
						IsArchiveEnabled: to.BoolPtr(true),
					},
					DockerFilePath: to.StringPtr("Dockerfile"),
					Platform: &armcontainerregistry.PlatformProperties{
						OS:           armcontainerregistry.OSLinux.ToPtr(),
						Architecture: armcontainerregistry.ArchitectureAmd64.ToPtr(),
					},
					ImageNames: []*string{
						to.StringPtr("testtaskrun:v1"),
					},
					IsPushEnabled:  to.BoolPtr(true),
					NoCache:        to.BoolPtr(false),
					SourceLocation: to.StringPtr("https://github.com/Azure-Samples/acr-build-helloworld-node.git"),
				},
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
	return &resp.TaskRun, nil
}

func getTaskRun(ctx context.Context, conn *arm.Connection) (*armcontainerregistry.TaskRun, error) {
	taskRunsClient := armcontainerregistry.NewTaskRunsClient(conn, subscriptionID)

	resp, err := taskRunsClient.Get(ctx, resourceGroupName, registryName, taskRunName, nil)
	if err != nil {
		return nil, err
	}
	return &resp.TaskRun, nil
}

func createResourceGroup(ctx context.Context, conn *arm.Connection) (*armresources.ResourceGroup, error) {
	resourceGroupClient := armresources.NewResourceGroupsClient(conn, subscriptionID)

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

func cleanup(ctx context.Context, conn *arm.Connection) (*http.Response, error) {
	resourceGroupClient := armresources.NewResourceGroupsClient(conn, subscriptionID)

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