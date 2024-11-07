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
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: HomeIcon
  },
  {
    type: backupprofile.Icon.briefcase,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: BriefcaseIcon
  },
  {
    type: backupprofile.Icon.fire,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: FireIcon
  },
  {
    type: backupprofile.Icon.envelope,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: EnvelopeIcon
  },
  {
    type: backupprofile.Icon.camera,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: CameraIcon
  },
  {
    type: backupprofile.Icon.book,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: BookOpenIcon
  },
];

export  function getIcon(icon: backupprofile.Icon): Icon {
  return icons.find(i => i.type === icon) ?? icons[0];
}