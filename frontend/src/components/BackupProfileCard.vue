<script setup lang='ts'>
import { useI18n } from "vue-i18n";
import { NoSymbolIcon, ShieldCheckIcon } from "@heroicons/vue/24/solid";
import { isAfter } from "@formkit/tempo";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref } from "vue";
import { Page, withId } from "../router";
import { useRouter } from "vue-router";
import { toCreationTimeBadge, toRepoTypeBadge } from "../common/badge";
import { toLongDateString, toRelativeTimeString } from "../common/time";
import BackupButton from "./BackupButton.vue";
import { repoStateChangedEvent } from "../common/events";
import { debounce } from "lodash";
import { getIcon, Icon } from "../common/icons";
import { getRepoType } from "../common/repository";
import * as backupClient from "../../bindings/github.com/loomi-labs/arco/backend/app/backupclient";
import * as repoClient from "../../bindings/github.com/loomi-labs/arco/backend/app/repositoryclient";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

import {Events} from "@wailsio/runtime";

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
const failedBackupRun = ref<string>("");
const icon = ref<Icon>(getIcon(props.backup.icon));

const bIds = props.backup.edges?.repositories?.filter((r) => r !== null)
  .map((r) => {
    const backupId = types.BackupId.createFrom();
    backupId.backupProfileId = props.backup.id;
    backupId.repositoryId = r.id;
    return backupId;
  }) ?? [];

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getFailedBackupRun() {
  for (const repoId of props.backup.edges?.repositories?.filter(r => r !== null).map((r) => r.id) ?? []) {
    try {
      const backupId = types.BackupId.createFrom();
      backupId.backupProfileId = props.backup.id;
      backupId.repositoryId = repoId;
      failedBackupRun.value = await backupClient.GetLastBackupErrorMsg(backupId);

      // We only care about the first error message.
      if (failedBackupRun.value) {
        break;
      }
    } catch (error: any) {
      await showAndLogError("Failed to get last backup error message", error);
    }
  }
}

async function getLastArchives() {
  try {
    let newLastArchive = undefined;
    for (const repo of props.backup.edges?.repositories?.filter(r => r !== null) ?? []) {
      const backupId = types.BackupId.createFrom();
      backupId.backupProfileId = props.backup.id;
      backupId.repositoryId = repo.id;
      const archive = await repoClient.GetLastArchiveByBackupId(backupId);
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

/************
 * Lifecycle
 ************/

getFailedBackupRun();
getLastArchives();

for (const backupId of bIds) {
  const handleRepoStateChanged = debounce(async () => {
    await getFailedBackupRun();
    await getLastArchives();
  }, 200);

  cleanupFunctions.push(Events.On(repoStateChangedEvent(backupId.repositoryId), handleRepoStateChanged));
}

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='group ac-card-hover h-full w-full'
       @click='router.push(withId(Page.BackupProfile, backup.id.toString()))'>
    <div
      class='flex justify-between rounded-t-lg bg-primary text-primary-content px-6 pt-4 pb-2 group-hover:bg-primary/50'>
      {{ props.backup.name }}
      <component :is='icon.html' class='size-8' />
    </div>
    <div class='flex justify-between items-center p-6'>
      <div class='w-full pr-6'>
        <!-- Info -->
        <div class='flex justify-between'>
          <p>{{ $t("last_backup") }}</p>
          <div>
            <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
              <span class='badge badge-error dark:border-error dark:text-error dark:bg-transparent truncate cursor-pointer'>{{ $t("failed") }}</span>
            </span>
            <span v-else-if='lastArchive' class='tooltip' :data-tip='toLongDateString(lastArchive.createdAt)'>
              <span :class='toCreationTimeBadge(lastArchive?.createdAt)'>{{ toRelativeTimeString(lastArchive.createdAt) }}</span>
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
          <ul class='text-right'>
            <li v-for='repo in props.backup.edges?.repositories?.filter(r => r !== null) ?? []' :key='repo.id'
                class='mx-1'
                :class='`${toRepoTypeBadge(getRepoType(repo.location))}`'
            >
              {{ repo.name }}
            </li>
          </ul>
        </div>
      </div>
      <BackupButton :backup-ids='bIds' />
    </div>
  </div>
</template>

<style scoped>

</style>