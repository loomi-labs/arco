<script setup lang='ts'>
import { useI18n } from "vue-i18n";
import { HomeIcon, NoSymbolIcon, ShieldCheckIcon } from "@heroicons/vue/24/solid";
import { borg, ent, state, types } from "../../wailsjs/go/models";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { isAfter } from "@formkit/tempo";
import { showAndLogError } from "../common/error";
import { ref } from "vue";
import { rBackupProfilePage, withId } from "../router";
import { useRouter } from "vue-router";
import { toDurationBadge } from "../common/badge";
import { toRelativeTimeString } from "../common/time";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as runtime from "../../wailsjs/runtime";
import { LogDebug } from "../../wailsjs/runtime";
import BackupButton from "./BackupButton.vue";
import { backupStateChangedEvent, repoStateChangedEvent } from "../common/events";

/************
 * Types
 ************/

interface Props {
  backup: ent.BackupProfile;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const { t } = useI18n();
const router = useRouter();
const lastArchive = ref<ent.Archive | undefined>(undefined);
const failedBackupRun = ref<string | undefined>(undefined);

const buttonStatus = ref<state.BackupButtonStatus | undefined>(undefined);
const backupProgress = ref<borg.BackupProgress | undefined>(undefined);

const bIds = props.backup.edges?.repositories?.map((r) => {
  const backupId = types.BackupId.createFrom();
  backupId.backupProfileId = props.backup.id;
  backupId.repositoryId = r.id;
  return backupId;
}) ?? [];

/************
 * Functions
 ************/

async function getFailedBackupRun() {
  for (const repoId of props.backup.edges?.repositories?.map((r) => r.id) ?? []) {
    try {
      const backupId = types.BackupId.createFrom();
      backupId.backupProfileId = props.backup.id;
      backupId.repositoryId = repoId;
      const repo = await repoClient.GetByBackupId(backupId);

      // We only care about the first failed backup run
      if (repo.edges.failedBackupRuns?.length) {
        failedBackupRun.value = repo.edges.failedBackupRuns?.[0]?.error;
        return;
      }
    } catch (error: any) {
      await showAndLogError("Failed to get repository", error);
    }
  }
}

async function getLastArchives() {
  try {
    let newLastArchive = undefined;
    for (const repo of props.backup.edges?.repositories ?? []) {
      const backupId = types.BackupId.createFrom();
      backupId.backupProfileId = props.backup.id;
      backupId.repositoryId = repo.id;
      const archive = await repoClient.GetLastArchive(backupId);
      if (archive?.id) {
        if (!newLastArchive || isAfter(archive.createdAt, newLastArchive.createdAt)) {
          newLastArchive = archive;
        }
      }
    }
    lastArchive.value = newLastArchive;
  } catch (error: any) {
    await showAndLogError(`Failed to get last archives for backup profile ${props.backup.id}`, error);
  }
}

async function getButtonStatus() {
  try {
    buttonStatus.value = await backupClient.GetCombinedBackupButtonStatus(bIds);
  } catch (error: any) {
    await showAndLogError("Failed to get backup state", error);
  }
}

async function getBackupProgress() {
  try {
    backupProgress.value = await backupClient.GetCombinedBackupProgress(bIds);
  } catch (error: any) {
    await showAndLogError("Failed to get backup progress", error);
  }
}

async function runBackups() {
  try {
    await backupClient.StartBackupJobs(props.backup.id);
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function abortBackups() {
  try {
    await backupClient.AbortBackupJobs(bIds);
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function runButtonAction() {
  if (buttonStatus.value === state.BackupButtonStatus.runBackup) {
    await runBackups();
  } else if (buttonStatus.value === state.BackupButtonStatus.abort) {
    await abortBackups();
  } else if (buttonStatus.value === state.BackupButtonStatus.locked) {
    // TODO: FIX THIS
    LogDebug("locked button");
  } else if (buttonStatus.value === state.BackupButtonStatus.unmount) {
    // TODO: FIX THIS
    LogDebug("unmount button");
  }
}

/************
 * Lifecycle
 ************/

getFailedBackupRun();
getLastArchives();
getButtonStatus();

for (const backupId of bIds) {
  runtime.EventsOn(backupStateChangedEvent(backupId), async () => {
    await getBackupProgress();
  });
  runtime.EventsOn(repoStateChangedEvent(backupId.repositoryId), async () => {
    await getButtonStatus();
    await getFailedBackupRun();
    await getLastArchives();
  });
}

</script>

<template>
  <div class='group/backup ac-card-hover h-full w-full'
       @click='router.push(withId(rBackupProfilePage, backup.id.toString()))'>
    <div
      class='flex justify-between bg-primary text-primary-content group-hover/backup:bg-primary/70 px-6 pt-4 pb-2'>
      {{ props.backup.name }}
      <HomeIcon class='size-8' />
    </div>
    <div class='flex justify-between items-center p-6'>
      <div class='w-full pr-6'>
        <!-- Info -->
        <div class='flex justify-between'>
          <p>{{ $t("last_backup") }}</p>
          <div>
            <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
              <span class='badge badge-outline badge-error'>{{ $t("failed") }}</span>
            </span>
            <span v-else-if='lastArchive' class='tooltip' :data-tip='lastArchive.createdAt'>
            <span :class='toDurationBadge(lastArchive?.createdAt)'>{{
                toRelativeTimeString(lastArchive.createdAt)
              }}</span>
          </span>
          </div>
        </div>
        <div class='divider'></div>
        <div class='flex justify-between'>
          <div>
            Automatic Backups
          </div>
          <div>
            <ShieldCheckIcon v-if='props.backup.edges.backupSchedule' class='size-6 text-success'></ShieldCheckIcon>
            <NoSymbolIcon v-else class='size-6 text-error'></NoSymbolIcon>
          </div>
        </div>
        <div class='divider'></div>
        <div class='flex justify-between items-center'>
          <div>
            Repositories
          </div>
          <div>
            <ul>
              <li v-for='(repo, index) in props.backup.edges?.repositories ?? []' :key='index'
                  class='badge badge-outline mx-1'>
                {{ repo.name }}
              </li>
            </ul>
          </div>
        </div>
      </div>
      <BackupButton :button-status='buttonStatus' :backup-progress='backupProgress' @click='runButtonAction' />
    </div>
  </div>
</template>

<style scoped>

</style>