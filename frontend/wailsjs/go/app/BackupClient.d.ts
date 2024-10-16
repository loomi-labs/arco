// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {types} from '../models';
import {ent} from '../models';
import {state} from '../models';
import {app} from '../models';
import {borg} from '../models';

export function AbortBackupJob(arg1:types.BackupId):Promise<void>;

export function AbortBackupJobs(arg1:Array<types.BackupId>):Promise<void>;

export function CreateBackupProfile(arg1:ent.BackupProfile,arg2:Array<number>):Promise<ent.BackupProfile>;

export function CreateDirectory(arg1:string):Promise<void>;

export function DeleteBackupProfile(arg1:number,arg2:boolean):Promise<void>;

export function DeleteBackupSchedule(arg1:number):Promise<void>;

export function DoesPathExist(arg1:string):Promise<boolean>;

export function DryRunPruneBackup(arg1:types.BackupId):Promise<void>;

export function DryRunPruneBackups(arg1:number):Promise<void>;

export function GetBackupButtonStatus(arg1:Array<types.BackupId>):Promise<state.BackupButtonStatus>;

export function GetBackupProfile(arg1:number):Promise<ent.BackupProfile>;

export function GetBackupProfileFilterOptions(arg1:number):Promise<Array<app.BackupProfileFilter>>;

export function GetBackupProfiles():Promise<Array<ent.BackupProfile>>;

export function GetCombinedBackupProgress(arg1:Array<types.BackupId>):Promise<borg.BackupProgress>;

export function GetDirectorySuggestions():Promise<Array<string>>;

export function GetPrefixSuggestion(arg1:string):Promise<string>;

export function GetState(arg1:types.BackupId):Promise<state.BackupState>;

export function IsDirectory(arg1:string):Promise<boolean>;

export function IsDirectoryEmpty(arg1:string):Promise<boolean>;

export function NewBackupProfile():Promise<ent.BackupProfile>;

export function PruneBackup(arg1:types.BackupId):Promise<void>;

export function PruneBackups(arg1:number):Promise<void>;

export function SaveBackupSchedule(arg1:number,arg2:ent.BackupSchedule):Promise<void>;

export function SelectDirectory():Promise<string>;

export function StartBackupJob(arg1:types.BackupId):Promise<void>;

export function StartBackupJobs(arg1:Array<types.BackupId>):Promise<Array<types.BackupId>>;

export function UpdateBackupProfile(arg1:ent.BackupProfile):Promise<ent.BackupProfile>;
