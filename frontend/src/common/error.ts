import { useToast } from "vue-toastification";
import * as appClient from "../../wailsjs/go/app/AppClient";

import { types } from "../../wailsjs/go/models";

const development = process.env.NODE_ENV === "development";

/**
 * showAndLogError shows an error message to the user and logs the error in the backend.
 * If the error is an instance of Error, it logs the error message and the stack trace.
 * If the error is a string, it logs the error message.
 * If the error is anything else, it logs "Unknown error".
 * @param message The error message to show to the user.
 * @param error The error to log.
 * @returns A promise that resolves when the error is logged.
 */
export async function showAndLogError(message: string, error?: any): Promise<void> {
  const toast = useToast();

  if (development) {
    toast.error(message + "\n" + error);
  } else {
    toast.error(message);
  }

  const fe = types.FrontendError.createFrom();

  // check type of error and log it in the backend
  if (error instanceof Error) {
    fe.message = error.message;
    if (error.stack) {
      fe.stack = error.stack;
    }
    await appClient.HandleError(message, fe);
  } else if (typeof error === "string") {
    fe.message = error.toString();
    await appClient.HandleError(message, fe);
  } else {
    fe.message = "Unknown error";
    await appClient.HandleError(message, fe);
  }
}