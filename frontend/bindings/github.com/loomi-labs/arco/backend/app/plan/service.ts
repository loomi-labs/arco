// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

/**
 * Service contains the business logic and provides methods exposed to the frontend
 * @module
 */

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Call as $Call, CancellablePromise as $CancellablePromise, Create as $Create } from "@wailsio/runtime";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as arcov1$0 from "../../api/v1/models.js";

/**
 * ListPlans returns available subscription plans
 */
export function ListPlans(): $CancellablePromise<(arcov1$0.Plan | null)[]> {
    return $Call.ByID(802962553).then(($result: any) => {
        return $$createType2($result);
    });
}

// Private type creation functions
const $$createType0 = arcov1$0.Plan.createFrom;
const $$createType1 = $Create.Nullable($$createType0);
const $$createType2 = $Create.Array($$createType1);
