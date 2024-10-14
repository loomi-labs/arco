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
    color: "bg-indigo-700 group-hover:bg-indigo-400/50 hover:bg-indigo-400/50 text-white",
    html: HomeIcon
  },
  {
    type: backupprofile.Icon.briefcase,
    color: "bg-purple-600 group-hover:bg-purple-600/50 hover:bg-purple-600/50 text-white",
    html: BriefcaseIcon
  },
  {
    type: backupprofile.Icon.fire,
    color: "bg-fuchsia-500 group-hover:bg-violet-500/50 hover:bg-violet-500/50 text-white",
    html: FireIcon
  },
  {
    type: backupprofile.Icon.envelope,
    color: "bg-blue-500 group-hover:bg-blue-500/50 hover:bg-blue-500/50 text-white",
    html: EnvelopeIcon
  },
  {
    type: backupprofile.Icon.camera,
    color: "bg-blue-700 group-hover:bg-blue-800/50 hover:bg-blue-800/50 text-white",
    html: CameraIcon
  },
  {
    type: backupprofile.Icon.book,
    color: "bg-sky-700 group-hover:bg-sky-400/50 hover:bg-sky-400/50 text-white",
    html: BookOpenIcon
  },
];

export  function getIcon(icon: backupprofile.Icon): Icon {
  return icons.find(i => i.type === icon) ?? icons[0];
}