// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

// AzClient is an interface for client to Azure
type AzClient interface {
	SetAuthorizer(authorizer autorest.Authorizer)
	SetSender(sender autorest.Sender)
	SetDuration(retryDuration time.Duration)

	ApplyRouteTable(string, string) error
	WaitForGetAccessOnGateway(maxRetryCount int) error
	GetGateway() (n.ApplicationGateway, error)
	UpdateGateway(*n.ApplicationGateway) error
	DeployGatewayWithVnet(ResourceGroup, ResourceName, ResourceName, string, string) error
	DeployGatewayWithSubnet(string, string) error
	GetSubnet(string) (n.Subnet, error)

	GetPublicIP(string) (n.PublicIPAddress, error)
}

type azClient struct {
	appGatewaysClient     n.ApplicationGatewaysClient
	publicIPsClient       n.PublicIPAddressesClient
	virtualNetworksClient n.VirtualNetworksClient
	subnetsClient         n.SubnetsClient
	routeTablesClient     n.RouteTablesClient
	groupsClient          r.GroupsClient
	deploymentsClient     r.DeploymentsClient
	clientID              string

	subscriptionID    SubscriptionID
	resourceGroupName ResourceGroup
	appGwName         ResourceName
	memoizedIPs       map[string]n.PublicIPAddress

	ctx context.Context
}

// NewAzClient returns an Azure Client
func NewAzClient(subscriptionID SubscriptionID, resourceGroupName ResourceGroup, appGwName ResourceName, uniqueUserAgentSuffix, clientID string) AzClient {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil
	}

	userAgent := fmt.Sprintf("ingress-appgw/%s/%s", version.Version, uniqueUserAgentSuffix)
	az := &azClient{
		appGatewaysClient:     n.NewApplicationGatewaysClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		publicIPsClient:       n.NewPublicIPAddressesClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		virtualNetworksClient: n.NewVirtualNetworksClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		subnetsClient:         n.NewSubnetsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		routeTablesClient:     n.NewRouteTablesClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		groupsClient:          r.NewGroupsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		deploymentsClient:     r.NewDeploymentsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		clientID:              clientID,

		subscriptionID:    subscriptionID,
		resourceGroupName: resourceGroupName,
		appGwName:         appGwName,
		memoizedIPs:       make(map[string]n.PublicIPAddress),

		ctx: context.Background(),
	}

	if err := az.appGatewaysClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to App Gateway client: ", userAgent)
	}
	if err := az.publicIPsClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to Public IP client: ", userAgent)
	}
	if err := az.virtualNetworksClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to Virtual Networks client: ", userAgent)
	}
	if err := az.subnetsClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to Subnets client: ", userAgent)
	}
	if err := az.routeTablesClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to Route Tables client: ", userAgent)
	}
	if err := az.groupsClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to Groups client: ", userAgent)
	}
	if err := az.deploymentsClient.AddToUserAgent(userAgent); err != nil {
		klog.Error("Error adding User Agent to Deployments client: ", userAgent)
	}

	// increase the polling duration to 60 minutes
	az.appGatewaysClient.PollingDuration = 60 * time.Minute
	az.deploymentsClient.PollingDuration = 60 * time.Minute

	return az
}

func (az *azClient) SetAuthorizer(authorizer autorest.Authorizer) {
	az.appGatewaysClient.Authorizer = authorizer
	az.publicIPsClient.Authorizer = authorizer
	az.virtualNetworksClient.Authorizer = authorizer
	az.subnetsClient.Authorizer = authorizer
	az.routeTablesClient.Authorizer = authorizer
	az.groupsClient.Authorizer = authorizer
	az.deploymentsClient.Authorizer = authorizer
}

func (az *azClient) SetSender(sender autorest.Sender) {
	az.appGatewaysClient.Client.Sender = sender
}

func (az *azClient) SetDuration(retryDuration time.Duration) {
	az.appGatewaysClient.Client.RetryDuration = retryDuration
}

func (az *azClient) WaitForGetAccessOnGateway(maxRetryCount int) (err error) {
	klog.V(3).Info("Getting Application Gateway configuration.")
	err = utils.Retry(maxRetryCount, retryPause,
		func() (utils.Retriable, error) {
			response, err := az.appGatewaysClient.Get(az.ctx, string(az.resourceGroupName), string(az.appGwName))
			if err == nil {
				return utils.Retriable(true), nil
			}

			e := controllererrors.NewErrorWithInnerErrorf(
				controllererrors.ErrorGetApplicationGatewayError,
				err,
				"Failed fetching configuration for Application Gateway. Will retry in %v.", retryPause,
			)

			if response.Response.Response != nil {
				e = controllererrors.NewErrorWithInnerErrorf(
					controllererrors.ErrorApplicationGatewayUnexpectedStatusCode,
					err,
					"Unexpected status code '%d' while performing a GET on Application Gateway.", response.Response.StatusCode,
				)

				if response.Response.StatusCode == 404 {
					e.Code = controllererrors.ErrorApplicationGatewayNotFound
				}

				if response.Response.StatusCode == 403 {
					e.Code = controllererrors.ErrorApplicationGatewayForbidden

					clientID := "<agic-client-id>"
					if az.clientID != "" {
						clientID = az.clientID
					}

					groupID := ResourceGroupID(az.subscriptionID, az.resourceGroupName)
					applicationGatewayID := ApplicationGatewayID(az.subscriptionID, az.resourceGroupName, az.appGwName)
					roleAssignmentCmd := fmt.Sprintf("az role assignment create --role Reader --scope %s --assignee %s;"+
						" az role assignment create --role Contributor --scope %s --assignee %s",
						groupID,
						clientID,
						applicationGatewayID,
						clientID,
					)

					e.Message += fmt.Sprintf(" You can use '%s' to assign permissions."+
						" AGIC Identity needs at least 'Contributor' access to Application Gateway '%s' and 'Reader' access to Application Gateway's Resource Group '%s'.",
						roleAssignmentCmd,
						string(az.appGwName),
						string(az.resourceGroupName),
					)
				}
				if response.Response.StatusCode == 400 || response.Response.StatusCode == 401 {
					klog.Errorf("configuration error (bad request) or unauthorized error while performing a GET using the authorizer")
					klog.Errorf("stopping GET retries")
					return utils.Retriable(false), e
				}
			}

			klog.Errorf(e.Error())
			if controllererrors.IsErrorCode(e, controllererrors.ErrorApplicationGatewayNotFound) {
				return utils.Retriable(false), e
			}

			return utils.Retriable(true), e
		})

	return
}

func (az *azClient) GetGateway() (gateway n.ApplicationGateway, err error) {
	err = utils.Retry(retryCount, retryPause,
		func() (utils.Retriable, error) {
			gateway, err = az.appGatewaysClient.Get(az.ctx, string(az.resourceGroupName), string(az.appGwName))
			if err != nil {
				klog.Errorf("Error while getting application gateway '%s': %s", string(az.appGwName), err)
			}
			return utils.Retriable(true), err
		})
	return
}

func (az *azClient) UpdateGateway(appGwObj *n.ApplicationGateway) (err error) {
	appGwFuture, err := az.appGatewaysClient.CreateOrUpdate(az.ctx, string(az.resourceGroupName), string(az.appGwName), *appGwObj)
	if err != nil {
		return
	}

	if appGwFuture.PollingURL() != "" {
		klog.V(3).Infof("OperationID='%s'", GetOperationIDFromPollingURL(appGwFuture.PollingURL()))
	}

	// Wait until deployment finshes and save the error message
	err = appGwFuture.WaitForCompletionRef(az.ctx, az.appGatewaysClient.BaseClient.Client)
	return
}

func (az *azClient) GetPublicIP(resourceID string) (n.PublicIPAddress, error) {
	if ip, ok := az.memoizedIPs[resourceID]; ok {
		return ip, nil
	}

	_, resourceGroupName, publicIPName := ParseResourceID(resourceID)

	ip, err := az.publicIPsClient.Get(az.ctx, string(resourceGroupName), string(publicIPName), "")
	if err != nil {
		return n.PublicIPAddress{}, err
	}
	az.memoizedIPs[resourceID] = ip
	return ip, nil
}

func (az *azClient) ApplyRouteTable(subnetID string, routeTableID string) error {
	// Check if the route table exists
	_, routeTableResourceGroup, routeTableName := ParseResourceID(routeTableID)
	routeTable, err := az.routeTablesClient.Get(az.ctx, string(routeTableResourceGroup), string(routeTableName), "")

	// if route table is not found, then simply add a log and return no error. routeTable will always be initialized.
	if routeTable.Response.StatusCode == 404 {
		klog.V(3).Infof("Error getting route table '%s' (this is relevant for AKS clusters using 'Kubenet' network plugin): %s",
			routeTableID,
			err.Error())
		return nil
	}

	if err != nil {
		// no access or no route table
		return err
	}

	// Get subnet and check if it is already associated to a route table
	_, subnetResourceGroup, subnetVnetName, subnetName := ParseSubResourceID(subnetID)
	subnet, err := az.subnetsClient.Get(az.ctx, string(subnetResourceGroup), string(subnetVnetName), string(subnetName), "")
	if err != nil {
		return err
	}

	if subnet.RouteTable != nil {
		if *subnet.RouteTable.ID != routeTableID {
			klog.V(3).Infof("Skipping associating Application Gateway subnet '%s' with route table '%s' used by k8s cluster as it is already associated to route table '%s'.",
				subnetID,
				routeTableID,
				*subnet.SubnetPropertiesFormat.RouteTable.ID)
		} else {
			klog.V(3).Infof("Application Gateway subnet '%s' is associated with route table '%s' used by k8s cluster.",
				subnetID,
				routeTableID)
		}

		return nil
	}

	klog.Infof("Associating Application Gateway subnet '%s' with route table '%s' used by k8s cluster.", subnetID, routeTableID)
	subnet.RouteTable = &routeTable

	subnetFuture, err := az.subnetsClient.CreateOrUpdate(az.ctx, string(subnetResourceGroup), string(subnetVnetName), string(subnetName), subnet)
	if err != nil {
		return err
	}

	// Wait until deployment finshes and save the error message
	err = subnetFuture.WaitForCompletionRef(az.ctx, az.subnetsClient.BaseClient.Client)
	if err != nil {
		return err
	}

	return nil
}

func (az *azClient) GetSubnet(subnetID string) (n.Subnet, error) {
	_, subnetResourceGroup, subnetVnetName, subnetName := ParseSubResourceID(subnetID)
	subnet, err := az.subnetsClient.Get(az.ctx, string(subnetResourceGroup), string(subnetVnetName), string(subnetName), "")
	return subnet, err
}

// DeployGatewayWithVnet creates Application Gateway within the specifid VNet. Implements AzClient interface.
func (az *azClient) DeployGatewayWithVnet(resourceGroupName ResourceGroup, vnetName ResourceName, subnetName ResourceName, subnetPrefix, skuName string) (err error) {
	vnet, err := az.getVnet(resourceGroupName, vnetName)
	if err != nil {
		return
	}

	klog.Infof("Checking the Vnet %s for a subnet with prefix %s", vnetName, subnetPrefix)
	subnet, err := az.findSubnet(vnet, subnetName, subnetPrefix)
	if err != nil {
		if subnetPrefix == "" {
			klog.Infof("Unable to find a subnet with subnetName %s. Please provide subnetPrefix in order to allow AGIC to create a subnet in Vnet %s", subnetName, vnetName)
			return
		}

		klog.Infof("Unable to find a subnet. Creating a subnet %s with prefix %s in Vnet %s", subnetName, subnetPrefix, vnetName)
		subnet, err = az.createSubnet(vnet, subnetName, subnetPrefix)
		if err != nil {
			return
		}
	}

	err = az.DeployGatewayWithSubnet(*subnet.ID, skuName)
	return
}

// DeployGatewayWithSubnet creates Application Gateway within the specifid subnet. Implements AzClient interface.
func (az *azClient) DeployGatewayWithSubnet(subnetID, skuName string) (err error) {
	klog.Infof("Deploying Gateway")

	// Check if group exists
	group, err := az.getGroup()
	if err != nil {
		return
	}
	klog.Infof("Using resource group: %v", *group.Name)

	deploymentName := string(az.appGwName)
	klog.Infof("Starting ARM template deployment: %s", deploymentName)
	result, err := az.createDeployment(subnetID, skuName)
	if err != nil {
		return
	}
	if result.Name != nil {
		klog.Infof("Completed deployment %v: %v", deploymentName, result.Properties.ProvisioningState)
	} else {
		klog.Infof("Completed deployment %v (no data returned to SDK)", deploymentName)
	}

	return
}

// Create a resource group for the deployment.
func (az *azClient) getGroup() (group r.Group, err error) {
	utils.Retry(retryCount, retryPause,
		func() (utils.Retriable, error) {
			group, err = az.groupsClient.Get(az.ctx, string(az.resourceGroupName))
			if err != nil {
				klog.Errorf("Error while getting resource group '%s': %s", az.resourceGroupName, err)
			}
			return utils.Retriable(true), err
		})

	return
}

func (az *azClient) getVnet(resourceGroupName ResourceGroup, vnetName ResourceName) (vnet n.VirtualNetwork, err error) {
	utils.Retry(extendedRetryCount, retryPause,
		func() (utils.Retriable, error) {
			vnet, err = az.virtualNetworksClient.Get(az.ctx, string(resourceGroupName), string(vnetName), "")
			if err != nil {
				klog.Errorf("Error while getting virtual network '%s': %s", vnetName, err)
			}
			return utils.Retriable(true), err
		})

	return
}

func (az *azClient) findSubnet(vnet n.VirtualNetwork, subnetName ResourceName, subnetPrefix string) (subnet n.Subnet, err error) {
	for _, subnet := range *vnet.Subnets {
		if string(subnetName) == *subnet.Name && (subnetPrefix == "" || subnetPrefix == *subnet.AddressPrefix) {
			return subnet, nil
		}
	}
	err = controllererrors.NewErrorf(
		controllererrors.ErrorSubnetNotFound,
		"Unable to find subnet with matching subnetName %s and subnetPrefix %s in virtual network %s", subnetName, subnetPrefix, *vnet.ID,
	)
	return
}

func (az *azClient) createSubnet(vnet n.VirtualNetwork, subnetName ResourceName, subnetPrefix string) (subnet n.Subnet, err error) {
	_, resourceGroup, vnetName := ParseResourceID(*vnet.ID)
	subnet = n.Subnet{
		SubnetPropertiesFormat: &n.SubnetPropertiesFormat{
			AddressPrefix: &subnetPrefix,
		},
	}
	subnetFuture, err := az.subnetsClient.CreateOrUpdate(az.ctx, string(resourceGroup), string(vnetName), string(subnetName), subnet)
	if err != nil {
		return
	}

	// Wait until deployment finshes and save the error message
	err = subnetFuture.WaitForCompletionRef(az.ctx, az.subnetsClient.BaseClient.Client)
	if err != nil {
		return
	}

	return az.subnetsClient.Get(az.ctx, string(resourceGroup), string(vnetName), string(subnetName), "")
}

// Create the deployment
func (az *azClient) createDeployment(subnetID, skuName string) (deployment r.DeploymentExtended, err error) {
	template := getTemplate()
	if err != nil {
		return
	}
	params := map[string]interface{}{
		"applicationGatewayName": map[string]string{
			"value": string(az.appGwName),
		},
		"applicationGatewaySubnetId": map[string]string{
			"value": subnetID,
		},
		"applicationGatewaySku": map[string]string{
			"value": skuName,
		},
	}

	deploymentFuture, err := az.deploymentsClient.CreateOrUpdate(
		az.ctx,
		string(az.resourceGroupName),
		string(az.appGwName),
		r.Deployment{
			Properties: &r.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       r.DeploymentModeIncremental,
			},
		},
	)
	if err != nil {
		return
	}
	err = deploymentFuture.WaitForCompletionRef(az.ctx, az.deploymentsClient.BaseClient.Client)
	if err != nil {
		return
	}
	return deploymentFuture.Result(az.deploymentsClient)
}

func getTemplate() map[string]interface{} {
	template := `
	{
		"$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		"contentVersion": "1.0.0.0",
		"parameters": {
			"applicationGatewayName": {
				"defaultValue": "appgw",
				"type": "string",
				"metadata": {
					"description": "Name of the Application Gateway."
				}
			},
			"applicationGatewaySubnetId": {
				"type": "string",
				"metadata": {
					"description": "Resource Id of Subnet in which Application Gateway will be deployed."
				}
			},
			"applicationGatewaySku": {
				"allowedValues": [
					"Standard_v2",
					"WAF_v2"
				],
				"type": "string",
				"metadata": {
					"description": "The sku of the Application Gateway. Default: WAF_v2 (Detection mode). In order to further customize WAF, use azure portal or cli."
				}
			}
		},
		"variables": {
			"resgpguid": "[substring(replace(guid(resourceGroup().id), '-', ''), 0, 4)]",
			"vnetName": "[concat('virtualnetwork' , variables('resgpguid'))]",
			"applicationGatewayPublicIpName": "[concat(parameters('applicationGatewayName'), '-appgwpip')]",
			"applicationGatewayPublicIpId": "[resourceId('Microsoft.Network/publicIPAddresses',variables('applicationGatewayPublicIpName'))]",
			"applicationGatewayId": "[resourceId('Microsoft.Network/applicationGateways', parameters('applicationGatewayName'))]",
			"webApplicationFirewallConfiguration": {
			  "enabled": "true",
			  "firewallMode": "Detection"
			}
		},
		"resources": [
			{
				"type": "Microsoft.Network/publicIPAddresses",
				"name": "[variables('applicationGatewayPublicIpName')]",
				"apiVersion": "2018-08-01",
				"location": "[resourceGroup().location]",
				"sku": {
					"name": "Standard"
				},
				"properties": {
					"publicIPAllocationMethod": "Static"
				}
			},
			{
				"type": "Microsoft.Network/applicationGateways",
				"name": "[parameters('applicationGatewayName')]",
				"apiVersion": "2018-08-01",
				"location": "[resourceGroup().location]",
				"tags": {
					"managed-by-k8s-ingress": "true",
					"created-by": "ingress-appgw"
				},
				"properties": {
					"sku": {
						"name": "[parameters('applicationGatewaySku')]",
						"tier": "[parameters('applicationGatewaySku')]",
						"capacity": 2
					},
					"gatewayIPConfigurations": [
						{
							"name": "appGatewayIpConfig",
							"properties": {
								"subnet": {
									"id": "[parameters('applicationGatewaySubnetId')]"
								}
							}
						}
					],
					"frontendIPConfigurations": [
						{
							"name": "appGatewayFrontendIP",
							"properties": {
								"PublicIPAddress": {
									"id": "[variables('applicationGatewayPublicIpId')]"
								}
							}
						}
					],
					"frontendPorts": [
						{
							"name": "httpPort",
							"properties": {
								"Port": 80
							}
						},
						{
							"name": "httpsPort",
							"properties": {
								"Port": 443
							}
						}
					],
					"backendAddressPools": [
						{
							"name": "bepool",
							"properties": {
								"backendAddresses": []
							}
						}
					],
					"httpListeners": [
						{
							"name": "httpListener",
							"properties": {
								"protocol": "Http",
								"frontendPort": {
									"id": "[concat(variables('applicationGatewayId'), '/frontendPorts/httpPort')]"
								},
								"frontendIPConfiguration": {
									"id": "[concat(variables('applicationGatewayId'), '/frontendIPConfigurations/appGatewayFrontendIP')]"
								}
							}
						}
					],
					"backendHttpSettingsCollection": [
						{
							"name": "setting",
							"properties": {
								"port": 80,
								"protocol": "Http"
							}
						}
					],
					"requestRoutingRules": [
						{
							"name": "rule1",
							"properties": {
								"httpListener": {
									"id": "[concat(variables('applicationGatewayId'), '/httpListeners/httpListener')]"
								},
								"backendAddressPool": {
									"id": "[concat(variables('applicationGatewayId'), '/backendAddressPools/bepool')]"
								},
								"backendHttpSettings": {
									"id": "[concat(variables('applicationGatewayId'), '/backendHttpSettingsCollection/setting')]"
								}
							}
						}
					],
					"webApplicationFirewallConfiguration": "[if(equals(parameters('applicationGatewaySku'), 'WAF_v2'), variables('webApplicationFirewallConfiguration'), json('null'))]"
				},
				"dependsOn": [
					"[concat('Microsoft.Network/publicIPAddresses/', variables('applicationGatewayPublicIpName'))]"
				]
			}
		],
		"outputs": {
			"subscriptionId": {
				"type": "string",
				"value": "[subscription().subscriptionId]"
			},
			"resourceGroupName": {
				"type": "string",
				"value": "[resourceGroup().name]"
			},
			"applicationGatewayName": {
				"type": "string",
				"value": "[parameters('applicationGatewayName')]"
			}
		}
	}`

	contents := make(map[string]interface{})
	json.Unmarshal([]byte(template), &contents)
	return contents
}
