// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package servicebus

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/resourcegroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/servicebus/2024-01-01/subscriptions"
	"github.com/hashicorp/go-azure-sdk/resource-manager/servicebus/2024-01-01/topics"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/servicebus/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func dataSourceServiceBusSubscription() *pluginsdk.Resource {
	resource := &pluginsdk.Resource{
		Read: dataSourceServiceBusSubscriptionRead,

		Timeouts: &pluginsdk.ResourceTimeout{
			Read: pluginsdk.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:     pluginsdk.TypeString,
				Required: true,
			},

			"topic_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: topics.ValidateTopicID,
			},

			"auto_delete_on_idle": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"default_message_ttl": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"lock_duration": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"dead_lettering_on_message_expiration": {
				Type:     pluginsdk.TypeBool,
				Computed: true,
			},

			"dead_lettering_on_filter_evaluation_error": {
				Type:     pluginsdk.TypeBool,
				Computed: true,
			},

			"batched_operations_enabled": {
				Type:     pluginsdk.TypeBool,
				Computed: true,
			},

			"max_delivery_count": {
				Type:     pluginsdk.TypeInt,
				Computed: true,
			},

			"requires_session": {
				Type:     pluginsdk.TypeBool,
				Computed: true,
			},

			"forward_to": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"forward_dead_lettered_messages_to": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},
		},
	}

	if !features.FivePointOh() {
		resource.Schema["topic_id"] = &pluginsdk.Schema{
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: topics.ValidateTopicID,
			AtLeastOneOf: []string{"topic_id", "topic_name", "resource_group_name", "namespace_name"},
		}

		resource.Schema["topic_name"] = &pluginsdk.Schema{
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validate.TopicName(),
			AtLeastOneOf: []string{"topic_id", "topic_name", "resource_group_name", "namespace_name"},
			Deprecated:   "`topic_name` will be removed in favour of the property `topic_id` in version 5.0 of the AzureRM Provider.",
		}

		resource.Schema["resource_group_name"] = &pluginsdk.Schema{
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: resourcegroups.ValidateName,
			AtLeastOneOf: []string{"topic_id", "topic_name", "resource_group_name", "namespace_name"},
			Deprecated:   "`resource_group_name` will be removed in favour of the property `topic_id` in version 5.0 of the AzureRM Provider.",
		}

		resource.Schema["namespace_name"] = &pluginsdk.Schema{
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: validate.NamespaceName,
			AtLeastOneOf: []string{"topic_id", "topic_name", "resource_group_name", "namespace_name"},
			Deprecated:   "`namespace_name` will be removed in favour of the property `topic_id` in version 5.0 of the AzureRM Provider.",
		}

		resource.Schema["enable_batched_operations"] = &pluginsdk.Schema{
			Type:     pluginsdk.TypeBool,
			Computed: true,
		}
	}

	return resource
}

func dataSourceServiceBusSubscriptionRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ServiceBus.SubscriptionsClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	var rgName string
	var nsName string
	var topicName string
	if v, ok := d.Get("topic_id").(string); ok && v != "" {
		topicId, err := subscriptions.ParseTopicID(v)
		if err != nil {
			return fmt.Errorf("parsing topic ID %q: %+v", v, err)
		}
		rgName = topicId.ResourceGroupName
		nsName = topicId.NamespaceName
		topicName = topicId.TopicName

		if !features.FivePointOh() && topicId.SubscriptionId == "" {
			rgName = d.Get("resource_group_name").(string)
			nsName = d.Get("namespace_name").(string)
			topicName = d.Get("topic_name").(string)
		}
	}

	id := subscriptions.NewSubscriptions2ID(subscriptionId, rgName, nsName, topicName, d.Get("name").(string))
	existing, err := client.Get(ctx, id)
	if err != nil {
		if response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("%s was not found", id)
		}

		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	d.SetId(id.ID())

	if model := existing.Model; model != nil {
		if props := model.Properties; props != nil {
			d.Set("auto_delete_on_idle", props.AutoDeleteOnIdle)
			d.Set("default_message_ttl", props.DefaultMessageTimeToLive)
			d.Set("lock_duration", props.LockDuration)
			d.Set("dead_lettering_on_message_expiration", props.DeadLetteringOnMessageExpiration)
			d.Set("dead_lettering_on_filter_evaluation_error", props.DeadLetteringOnFilterEvaluationExceptions)
			d.Set("batched_operations_enabled", props.EnableBatchedOperations)
			d.Set("requires_session", props.RequiresSession)
			d.Set("forward_dead_lettered_messages_to", props.ForwardDeadLetteredMessagesTo)
			d.Set("forward_to", props.ForwardTo)

			maxDeliveryCount := 0
			if props.MaxDeliveryCount != nil {
				maxDeliveryCount = int(*props.MaxDeliveryCount)
			}

			d.Set("max_delivery_count", maxDeliveryCount)

			if !features.FivePointOh() {
				d.Set("enable_batched_operations", props.EnableBatchedOperations)
			}
		}
	}

	return nil
}
