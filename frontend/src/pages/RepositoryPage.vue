<script setup lang='ts'>
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent, state } from "../../wailsjs/go/models";
import { nextTick, ref, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { showAndLogError } from "../common/error";
import { PencilIcon } from "@heroicons/vue/24/solid";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/zod";
import * as zod from "zod";
import { object } from "zod";
import { getBadgeColor, getLocation, getTextColor, Location, toHumanReadableSize } from "../common/repository";
import { toDurationBadge } from "../common/badge";
import { toRelativeTimeString } from "../common/time";
import ArchivesCard from "../components/ArchivesCard.vue";

/************
 * Variables
 ************/

const router = useRouter();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const loading = ref(true);
const location = ref<Location>(Location.Local);
const nbrOfArchives = ref<number>(0);
const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");
const lastArchive = ref<ent.Archive | undefined>(undefined);
const failedBackupRun = ref<string | undefined>(undefined);

const nameInputKey = "name_input";
const nameInput = useTemplateRef<InstanceType<typeof HTMLInputElement>>(nameInputKey);

const { meta, errors, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ required_error: "Enter a name for this repository" })
        .min(3, { message: "Name length must be at least 3" })
        .max(30, { message: "Name is too long" })
    })
  )
});

const [name, nameAttrs] = defineField("name", { validateOnBlur: false });

/************
 * Functions
 ************/

async function getRepo() {
  try {
    loading.value = true;
    const repoId = parseInt(router.currentRoute.value.params.id as string);
    repo.value = await repoClient.Get(repoId);
    name.value = repo.value.name;

    location.value = getLocation(repo.value.location);

    repoState.value = await repoClient.GetState(repoId);

    nbrOfArchives.value = await repoClient.GetNbrOfArchives(repoId);

    totalSize.value = toHumanReadableSize(repo.value.stats_total_size);
    sizeOnDisk.value = toHumanReadableSize(repo.value.stats_unique_csize);
    failedBackupRun.value = repo.value.edges.failedBackupRuns?.[0]?.error;

    lastArchive.value = await repoClient.GetLastArchiveByRepoId(repoId);
  } catch (error: any) {
    await showAndLogError("Failed to get repository", error);
  }
  loading.value = false;
}

async function saveName() {
  if (meta.value.valid && name.value !== repo.value.name) {
    try {
      repo.value.name = name.value ?? "";
      await repoClient.Update(repo.value);
    } catch (error: any) {
      await showAndLogError("Failed to save repository name", error);
    }
  }
}

function adjustNameWidth() {
  if (nameInput.value) {
    nameInput.value.style.width = "30px";
    nameInput.value.style.width = `${nameInput.value.scrollWidth}px`;
  }
}

/************
 * Lifecycle
 ************/

getRepo();

watch(loading, async () => {
  // Wait for the loading to finish before adjusting the name width
  await nextTick();
  adjustNameWidth();
});

</script>

<template>
  <div v-if='loading' class='flex items-center justify-center min-h-svh'>
    <div class='loading loading-ring loading-lg'></div>
  </div>
  <div v-else class='container mx-auto text-left pt-10'>
    <!-- Data Section -->
    <div class='flex items-center justify-between mb-4'>
      <!-- Name -->
      <label class='flex items-center gap-2'>
        <input :ref='nameInputKey'
               type='text'
               class='text-2xl font-bold bg-transparent w-10'
               :class='getTextColor(location)'
               v-model='name'
               v-bind='nameAttrs'
               @change='saveName'
               @input='adjustNameWidth'
        />
        <PencilIcon class='size-4' :class='getTextColor(location)' />
        <span class='text-error'>{{ errors.name }}</span>
      </label>
    </div>

    <div class='flex flex-col w-full py-6'>
      <div class='flex justify-between'>
        <div>{{ $t("archives") }}</div>
        <div>{{ nbrOfArchives }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("location") }}</div>
        <div class='flex items-center gap-4'>
          <span>{{ repo.location }}</span>
          <span class='badge badge-outline'
                :class='getBadgeColor(location)'>{{ location === Location.Local ? $t("local") : $t("remote") }}</span>
        </div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("last_backup") }}</div>
        <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
            <span class='badge badge-outline badge-error'>{{ $t("failed") }}</span>
          </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='lastArchive.createdAt'>
            <span :class='toDurationBadge(lastArchive?.createdAt)'>{{ toRelativeTimeString(lastArchive.createdAt)
              }}</span>
          </span>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("total_size") }}</div>
        <div>{{ totalSize }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("size_on_disk") }}</div>
        <div>{{ sizeOnDisk }}</div>
      </div>
    </div>

    <ArchivesCard :repo='repo'
                  :repo-status='repoState.status'
                  :highlight='false'>
    </ArchivesCard>
  </div>
</template>

<style scoped>

</style>