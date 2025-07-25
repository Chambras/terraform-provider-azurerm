// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/web/2023-12-01/webapps"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
)

const (
	ServicePlanTypeAppPlan         = "app"
	ServicePlanTypeConsumption     = "consumption"
	ServicePlanTypeElastic         = "elastic"
	ServicePlanTypeFlexConsumption = "flexconsumption"
	ServicePlanTypeIsolated        = "isolated"
	ServicePlanTypePremium         = "premium"
	ServicePlanTypeWorkflow        = "workflow"
)

var appServicePlanSkus = []string{
	"B1", "B2", "B3", // basic
	"S1", "S2", "S3", // standard
}

var freeSkus = []string{
	"F1",
}

var sharedSkus = []string{
	"D1",
	"SHARED",
}

var consumptionSkus = []string{
	"Y1",
}

var premiumSkus = []string{
	"P1v2", "P2v2", "P3v2", // Premium V2
	"P0v3", "P1v3", "P2v3", "P3v3", // Premium V3
	"P1mv3", "P2mv3", "P3mv3", "P4mv3", "P5mv3", // Premium V3 memory optimized
}

var flexConsumptionSkus = []string{
	"FC1",
}

var elasticSkus = []string{
	"EP1", "EP2", "EP3",
}

var isolatedSkus = []string{
	"I1", "I2", "I3", // Isolated V1 - ASEV2
	"I1v2", "I2v2", "I3v2", "I4v2", "I5v2", "I6v2", // Isolated v2 - ASEv3
	"I1mv2", "I2mv2", "I3mv2", "I4mv2", "I5mv2", // Isolated v2 - ASEv3 memory optimized
}

var workflowSkus = []string{
	"WS1", "WS2", "WS3",
}

// AllKnownServicePlanSkus returns a list of all supported known SKU names
func AllKnownServicePlanSkus() []string {
	allSkus := make([]string, 0)
	allSkus = append(allSkus, appServicePlanSkus...)
	allSkus = append(allSkus, consumptionSkus...)
	allSkus = append(allSkus, elasticSkus...)
	allSkus = append(allSkus, flexConsumptionSkus...)
	allSkus = append(allSkus, freeSkus...)
	allSkus = append(allSkus, isolatedSkus...)
	allSkus = append(allSkus, premiumSkus...)
	allSkus = append(allSkus, sharedSkus...)
	allSkus = append(allSkus, workflowSkus...)

	return allSkus
}

func PlanIsConsumption(input *string) bool {
	if input == nil {
		return false
	}
	for _, v := range consumptionSkus {
		if strings.EqualFold(*input, v) {
			return true
		}
	}

	return false
}

func PlanIsPremium(input string) bool {
	if input == "" {
		return false
	}
	for _, v := range premiumSkus {
		if strings.EqualFold(input, v) {
			return true
		}
	}

	return false
}

func PlanIsFlexConsumption(input *string) bool {
	if input == nil {
		return false
	}
	for _, v := range flexConsumptionSkus {
		if strings.EqualFold(*input, v) {
			return true
		}
	}

	return false
}

func PlanIsElastic(input *string) bool {
	if input == nil {
		return false
	}
	for _, v := range elasticSkus {
		if strings.EqualFold(*input, v) {
			return true
		}
	}

	return false
}

func PlanIsIsolated(input *string) bool {
	if input == nil {
		return false
	}
	for _, v := range isolatedSkus {
		if strings.EqualFold(*input, v) {
			return true
		}
	}

	return false
}

func PlanIsAppPlan(input *string) bool {
	if input == nil {
		return false
	}
	for _, v := range appServicePlanSkus {
		if strings.EqualFold(*input, v) {
			return true
		}
	}

	return false
}

func PlanIsWorkflow(input *string) bool {
	if input == nil {
		return false
	}
	for _, v := range workflowSkus {
		if strings.EqualFold(*input, v) {
			return true
		}
	}

	return false
}

func PlanTypeFromSku(input string) string {
	if PlanIsPremium(input) {
		return ServicePlanTypePremium
	}

	if PlanIsWorkflow(&input) {
		return ServicePlanTypeWorkflow
	}

	if PlanIsConsumption(&input) {
		return ServicePlanTypeConsumption
	}

	if PlanIsFlexConsumption(&input) {
		return ServicePlanTypeFlexConsumption
	}

	if PlanIsElastic(&input) {
		return ServicePlanTypeElastic
	}

	if PlanIsIsolated(&input) {
		return ServicePlanTypeIsolated
	}

	if PlanIsAppPlan(&input) {
		return ServicePlanTypeAppPlan
	}

	return "unknown"
}

func PlanSupportsZoneBalancing(input string) bool {
	switch PlanTypeFromSku(input) {
	case ServicePlanTypePremium, ServicePlanTypeElastic, ServicePlanTypeWorkflow, ServicePlanTypeConsumption, ServicePlanTypeFlexConsumption, ServicePlanTypeIsolated:
		return true
	default:
		return false
	}
}

func PlanSupportsScaleOut(plan string) bool {
	return strings.HasPrefix(plan, "EP") || strings.HasPrefix(plan, "WS")
}

// ServicePlanInfoForApp returns the OS type and Service Plan SKU for a given App Service Resource
func ServicePlanInfoForApp(ctx context.Context, metadata sdk.ResourceMetaData, id commonids.AppServiceId) (osType *string, planSku *string, err error) {
	client := metadata.Client.AppService.WebAppsClient
	servicePlanClient := metadata.Client.AppService.ServicePlanClient

	site, err := client.Get(ctx, id)
	if err != nil || site.Model == nil || site.Model.Properties == nil {
		return nil, nil, fmt.Errorf("retrieving %s: %+v", id, err)
	}
	props := *site.Model.Properties
	if props.ServerFarmId == nil {
		return nil, nil, fmt.Errorf("determining Service Plan ID for %s: %+v", id, err)
	}
	servicePlanId, err := commonids.ParseAppServicePlanIDInsensitively(*props.ServerFarmId)
	if err != nil {
		return nil, nil, err
	}

	sp, err := servicePlanClient.Get(ctx, *servicePlanId)
	if err != nil || sp.Model.Kind == nil {
		return nil, nil, fmt.Errorf("retrieving Service Plan for %s: %+v", id, err)
	}

	osType = pointer.To("windows")
	if strings.Contains(strings.ToLower(*sp.Model.Kind), "linux") {
		osType = pointer.To("linux")
	}

	planSku = pointer.To("")
	if sku := sp.Model.Sku; sku != nil {
		planSku = sku.Name
	}

	return osType, planSku, nil
}

// ServicePlanInfoForAppSlot returns the OS type and Service Plan SKU for a given App Service Resource
func ServicePlanInfoForAppSlot(ctx context.Context, metadata sdk.ResourceMetaData, id webapps.SlotId) (osType *string, planSku *string, err error) {
	client := metadata.Client.AppService.WebAppsClient
	servicePlanClient := metadata.Client.AppService.ServicePlanClient

	site, err := client.GetSlot(ctx, id)
	if err != nil || site.Model == nil || site.Model.Properties == nil {
		return nil, nil, fmt.Errorf("retrieving %s: %+v", id, err)
	}
	props := *site.Model.Properties
	if props.ServerFarmId == nil {
		return nil, nil, fmt.Errorf("determining Service Plan ID for %s: %+v", id, err)
	}
	servicePlanId, err := commonids.ParseAppServicePlanIDInsensitively(*props.ServerFarmId)
	if err != nil {
		return nil, nil, err
	}

	sp, err := servicePlanClient.Get(ctx, *servicePlanId)
	if err != nil || sp.Model.Kind == nil {
		return nil, nil, fmt.Errorf("retrieving Service Plan for %s: %+v", id, err)
	}

	osType = pointer.To("windows")
	if strings.Contains(strings.ToLower(*sp.Model.Kind), "linux") {
		osType = pointer.To("linux")
	}

	planSku = pointer.To("")
	if sku := sp.Model.Sku; sku != nil {
		planSku = sku.Name
	}

	return osType, planSku, nil
}
