import { useToast } from "vue-toastification";

import * as appClient from "../../bindings/github.com/loomi-labs/arco/backend/app/appclient";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";



const development = process.env.NODE_ENV === "development";

/**
 * createFrontendError creates a FrontendError from various error types
 */
function createFrontendError(error?: unknown): types.FrontendError {
  const fe = types.FrontendError.createFrom();
  
  if (error instanceof Error) {
    fe.message = error.message;
    if (error.stack) {
      fe.stack = error.stack;
    }
  } else if (typeof error === "string") {
    fe.message = error.toString();
  } else {
    fe.message = "Unknown error";
  }
  
  return fe;
}

/**
 * showAndLogError shows an error message to the user and logs the error in the backend.
 * If the error is an instance of Error, it logs the error message and the stack trace.
 * If the error is a string, it logs the error message.
 * If the error is anything else, it logs "Unknown error".
 * @param message The error message to show to the user.
 * @param error The error to log.
 * @returns A promise that resolves when the error is logged.
 */
export async function showAndLogError(message: string, error?: unknown): Promise<void> {
  const toast = useToast();

  if (development) {
    toast.error(message + "\n" + error);
  } else {
    toast.error(message);
  }

  const fe = createFrontendError(error);
  await appClient.HandleError(message, fe);
}

/**
 * logError logs an error to the backend without showing a toast notification.
 * Use this when you want to display the error in the UI and log it for debugging.
 * @param message The error context message.
 * @param error The error to log.
 * @returns A promise that resolves when the error is logged.
 */
export async function logError(message: string, error?: unknown): Promise<void> {
  const fe = createFrontendError(error);
  await appClient.HandleError(message, fe);
}

/**
 * logDebug logs a debug message to the backend.
 * @param message The debug message to log.
 * @returns A promise that resolves when the message is logged.
 */
export async function logDebug(message: string): Promise<void> {
  try {
    await appClient.LogDebug(message);
  } catch (_error) {
    // Ignore logging errors to prevent infinite recursion
  }
}