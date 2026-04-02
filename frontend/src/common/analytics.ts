import router from "../router";
import { logError } from "./logger";
import * as analyticsService from "../../bindings/github.com/loomi-labs/arco/backend/app/analytics/service";
import { EventName } from "../../bindings/github.com/loomi-labs/arco/backend/app/analytics/models";

// Sanitize route paths by replacing dynamic IDs with :id
function sanitizePath(path: string): string {
  return path.replace(/\/\d+/g, "/:id");
}

// Setup page view tracking via router afterEach guard
export function setupPageViewTracking(): () => void {
  const removeGuard = router.afterEach(async (to) => {
    const page = sanitizePath(to.path);
    try {
      await analyticsService.TrackEvent(EventName.EventPageView, { page });
    } catch (error: unknown) {
      await logError("Failed to track page view", error);
    }
  });

  return removeGuard;
}
