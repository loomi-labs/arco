<script setup lang='ts'>
import { useI18n } from "vue-i18n";
import { HomeIcon, ShieldCheckIcon, NoSymbolIcon } from "@heroicons/vue/24/solid";
import { ent, types } from "../../wailsjs/go/models";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { isAfter } from "@formkit/tempo";
import { showAndLogError } from "../common/error";
import { ref } from "vue";
import { rDataDetailPage, withId } from "../router";
import { useRouter } from "vue-router";
import { getBadgeStyle } from "../common/badge";
import { toRelativeTimeString } from "../common/time";

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
      if (repo.edges.failed_backup_runs?.length) {
        failedBackupRun.value = repo.edges.failed_backup_runs?.[0]?.error;
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
      if (archive.id) {
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

</script>

<template>
  <div class='group/backup bg-base-100 hover:bg-base-100/50 rounded-xl shadow-lg h-full w-full cursor-pointer'
       @click='router.push(withId(rDataDetailPage, backup.id.toString()))'>
    <div
      class='flex justify-between bg-primary text-primary-content group-hover/backup:bg-primary/70 rounded-t-xl px-6 pt-4 pb-2'>
      {{ props.backup.name }}
      <HomeIcon class='size-8' />
    </div>
    <div class='flex justify-between items-center px-6 pt-6'>
      <div class='w-full pr-6'>
        <!-- Info -->
        <div class='flex justify-between'>
          <p>{{ $t("last_backup") }}</p>
          <div>
            <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
              <span class='badge badge-outline badge-error'>{{ $t("failed") }}</span>
            </span>
            <span v-else-if='lastArchive' class='tooltip' :data-tip='lastArchive.createdAt'>
            <span :class='getBadgeStyle(lastArchive?.createdAt)'>{{
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
              <li v-for='(repo, index) in props.backup.edges?.repositories ?? []' :key='index' class='badge badge-outline mx-1'>
                {{ repo.name }}
              </li>
            </ul>
          </div>
        </div>
      </div>
      <!-- Button -->
      <div class='stack'>
        <div class='flex items-center justify-center w-[94px] h-[94px]'>
          <button class='btn btn-circle p-4 m-0 w-16 h-16'
          >The Button
          </button>
        </div>
        <div class='relative'>
          <div
            class='radial-progress absolute bottom-[2px] left-0'
            :style='`--value:100; --size:95px; --thickness: 6px;`'
            role='progressbar'>
          </div>
        </div>
      </div>
    </div>
  </div>

</template>

<style scoped>

</style>