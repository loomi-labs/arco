import { BookOpenIcon, BriefcaseIcon, CameraIcon, EnvelopeIcon, FireIcon, HomeIcon } from "@heroicons/vue/24/solid";
import * as backupprofile from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";


export interface Icon {
  type: backupprofile.Icon;
  color: string;
  html: any;
}

export const icons: Icon[] = [
  {
    type: backupprofile.Icon.IconHome,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: HomeIcon
  },
  {
    type: backupprofile.Icon.IconBriefcase,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: BriefcaseIcon
  },
  {
    type: backupprofile.Icon.IconFire,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: FireIcon
  },
  {
    type: backupprofile.Icon.IconEnvelope,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: EnvelopeIcon
  },
  {
    type: backupprofile.Icon.IconCamera,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: CameraIcon
  },
  {
    type: backupprofile.Icon.IconBook,
    color: "bg-primary text-primary-content hover:bg-primary/50",
    html: BookOpenIcon
  },
];

export  function getIcon(icon: backupprofile.Icon): Icon {
  return icons.find(i => i.type === icon) ?? icons[0];
}