import { backupprofile } from "../../wailsjs/go/models";
import { BookOpenIcon, BriefcaseIcon, CameraIcon, EnvelopeIcon, FireIcon, HomeIcon } from "@heroicons/vue/24/solid";

export interface Icon {
  type: backupprofile.Icon;
  color: string;
  html: any;
}

export const icons: Icon[] = [
  {
    type: backupprofile.Icon.home,
    color: "bg-blue-500 group-hover:bg-blue-500/50 text-dark dark:text-white",
    html: HomeIcon
  },
  {
    type: backupprofile.Icon.briefcase,
    color: "bg-indigo-500 group-hover:bg-indigo-500/50 text-dark dark:text-white",
    html: BriefcaseIcon
  },
  {
    type: backupprofile.Icon.book,
    color: "bg-purple-500 group-hover:bg-purple-500/50 text-dark dark:text-white",
    html: BookOpenIcon
  },
  {
    type: backupprofile.Icon.envelope,
    color: "bg-green-500 group-hover:bg-green-500/50 text-dark dark:text-white",
    html: EnvelopeIcon
  },
  {
    type: backupprofile.Icon.camera,
    color: "bg-yellow-500 group-hover:bg-yellow-500/50 text-dark dark:text-white",
    html: CameraIcon
  },
  {
    type: backupprofile.Icon.fire,
    color: "bg-red-500 group-hover:bg-red-500/50 text-dark dark:text-white",
    html: FireIcon
  }
];

export  function getIcon(icon: backupprofile.Icon): Icon {
  return icons.find(i => i.type === icon) ?? icons[0];
}