import { useToast } from "vue-toastification";
import * as appClient from "../../wailsjs/go/app/AppClient";

import { types } from "../../wailsjs/go/models";

const development = process.env.NODE_ENV === "development";

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