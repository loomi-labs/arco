import { FeatureSet } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";

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
 * Get features list based on plan feature set
 */
export function getFeaturesByPlan(featureSet: FeatureSet | undefined): PlanFeature[] {
  const isPro = featureSet === FeatureSet.FeatureSet_FEATURE_SET_PRO;
  
  return [
    {
      text: "Cloud backup storage"
    },
    {
      text: "Secure encrypted backups"
    },
    {
      text: `Retention of ${isPro ? '60' : '30'} days`,
      highlight: isPro
    }
  ];
}

/**
 * Get retention period in days for a plan
 */
export function getRetentionDays(featureSet: FeatureSet | undefined): number {
  return featureSet === FeatureSet.FeatureSet_FEATURE_SET_PRO ? 60 : 30;
}

/**
 * Get plan display name
 */
export function getPlanDisplayName(featureSet: FeatureSet | undefined): string {
  return featureSet === FeatureSet.FeatureSet_FEATURE_SET_PRO ? 'Pro' : 'Basic';
}