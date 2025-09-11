import type { Plan } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";

/************
 * Types
 ************/

export interface PlanFeature {
  text: string;
  highlight?: boolean;
}

/************
 * Functions
 ************/

/**
 * Get features list based on plan details
 */
export function getFeaturesByPlan(plan: Plan | undefined): PlanFeature[] {
  if (!plan) {
    return [
      {
        text: "Cloud backup storage"
      },
      {
        text: "Secure encrypted backups"
      },
      {
        text: "Retention of 30 days"
      }
    ];
  }

  // Determine if this is a Pro plan based on having overage pricing
  const isProPlan = (plan.overage_cents_per_gb ?? 0) > 0;
  
  return [
    {
      text: "Cloud backup storage"
    },
    {
      text: "Secure encrypted backups"
    },
    {
      text: `${plan.storage_gb ?? 0}GB storage included`
    },
    {
      text: `Up to ${plan.allowed_repositories ?? 0} repositories`
    },
    ...(isProPlan ? [{
      text: "Overage pricing beyond base storage",
      highlight: true
    }] : [])
  ];
}

/**
 * Get retention period in days for a plan
 */
export function getRetentionDays(plan: Plan | undefined): number {
  // For now, return default retention based on plan features
  // This could be moved to the proto if needed
  const isProPlan = (plan?.overage_cents_per_gb ?? 0) > 0;
  return isProPlan ? 60 : 30;
}

/**
 * Get plan display name from plan object
 */
export function getPlanDisplayName(plan: Plan | undefined): string {
  return plan?.name ?? 'Unknown Plan';
}