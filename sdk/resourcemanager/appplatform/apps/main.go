// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package main

import (
	"context"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appplatform/armappplatform"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	subscriptionID    string
	location          = "westus"
	resourceGroupName = "sample-resource-group"
	serviceName       = "sample-spring-cloud"
	appName           = "sample-app"
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

	service, err := createSpringCloudService(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("app platform service:", *service.ID)

	app, err := createAPP(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("spring cloud app:", *app.ID)

	app, err = getAppResource(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("get spring cloud app:", *app.ID)

	uploadURL, err := getAppResourceUploadURL(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("app resource upload url:", *uploadURL.RelativePath, *uploadURL.UploadURL)

	domain, err := validateDomain(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("validate domain:", *domain.IsValid, *domain.Message)

	keepResource := os.Getenv("KEEP_RESOURCE")
	if len(keepResource) == 0 {
		err = cleanup(ctx, cred)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("cleaned up successfully.")
	}
}

func createSpringCloudService(ctx context.Context, cred azcore.TokenCredential) (*armappplatform.ServiceResource, error) {
	servicesClient, err := armappplatform.NewServicesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	pollerResp, err := servicesClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		serviceName,
		armappplatform.ServiceResource{
			Location: to.Ptr(location),
			SKU: &armappplatform.SKU{
				Name: to.Ptr("S0"),
				Tier: to.Ptr("Standard"),
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.ServiceResource, nil
}

func createAPP(ctx context.Context, cred azcore.TokenCredential) (*armappplatform.AppResource, error) {
	appsClient, err := armappplatform.NewAppsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	pollerResp, err := appsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		serviceName,
		appName,
		armappplatform.AppResource{
			Properties: &armappplatform.AppResourceProperties{
				Public:    to.Ptr(true),
				HTTPSOnly: to.Ptr(false),
				TemporaryDisk: &armappplatform.TemporaryDisk{
					SizeInGB:  to.Ptr[int32](1),
					MountPath: to.Ptr("/mytemporary"),
				},
				PersistentDisk: &armappplatform.PersistentDisk{
					SizeInGB:  to.Ptr[int32](1),
					MountPath: to.Ptr("/mypersistent"),
				},
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.AppResource, nil
}

func getAppResource(ctx context.Context, cred azcore.TokenCredential) (*armappplatform.AppResource, error) {
	appsClient, err := armappplatform.NewAppsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resp, err := appsClient.Get(ctx, resourceGroupName, serviceName, appName, nil)
	if err != nil {
		return nil, err
	}

	return &resp.AppResource, nil
}

func getAppResourceUploadURL(ctx context.Context, cred azcore.TokenCredential) (*armappplatform.ResourceUploadDefinition, error) {
	appsClient, err := armappplatform.NewAppsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resp, err := appsClient.GetResourceUploadURL(ctx, resourceGroupName, serviceName, appName, nil)
	if err != nil {
		return nil, err
	}

	return &resp.ResourceUploadDefinition, nil
}

func validateDomain(ctx context.Context, cred azcore.TokenCredential) (*armappplatform.CustomDomainValidateResult, error) {
	appsClient, err := armappplatform.NewAppsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resp, err := appsClient.ValidateDomain(
		ctx,
		resourceGroupName,
		serviceName,
		appName,
		armappplatform.CustomDomainValidatePayload{
			Name: to.Ptr("test"),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &resp.CustomDomainValidateResult, nil
}

func createResourceGroup(ctx context.Context, cred azcore.TokenCredential) (*armresources.ResourceGroup, error) {
	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	resourceGroupResp, err := resourceGroupClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		armresources.ResourceGroup{
			Location: to.Ptr(location),
		},
		nil)
	if err != nil {
		return nil, err
	}

	return &resourceGroupResp.ResourceGroup, nil
}

func cleanup(ctx context.Context, cred azcore.TokenCredential) error {
	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return err
	}

	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
