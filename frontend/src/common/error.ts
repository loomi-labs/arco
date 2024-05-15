import { useToast } from "vue-toastification";
import { HandleError } from "../../wailsjs/go/borg/Borg";
import { borg } from "../../wailsjs/go/models";

const development = process.env.NODE_ENV === "development";

export async function showAndLogError(message: string, error?: any): Promise<void> {
  const toast = useToast();

  if (development) {
    toast.error(message + " " + error);
  } else {
    toast.error(message);
  }

  const fe = borg.FrontendError.createFrom();

  // check type of error and log it in the backend
  if (error instanceof Error) {
    fe.message = error.message;
    if (error.stack) {
      fe.stack = error.stack;
    }
    await HandleError(message, fe);
  } else if (typeof error === "string") {
    fe.message = error.toString();
    await HandleError(message, fe);
  } else {
    fe.message = "Unknown error";
    await HandleError(message, fe);
  }
}